package quayclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"resty.dev/v3"
)

const (
	apiBaseURL = "https://quay.io"
	pageSize   = 100
)

type Client struct {
	client    *resty.Client
	namespace string
	token     string
}

// Repository represents a repository (image) on Quay.
type Repository struct {
	Name      string
	Namespace string
	IsPrivate bool
}

type quayRepository struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	IsPublic  bool   `json:"is_public"`
	Kind      string `json:"kind"`
}

type listResponse struct {
	Repositories  []quayRepository `json:"repositories"`
	HasAdditional bool             `json:"has_additional"`
}

// New returns a client for the Quay API. Without a token only public
// repositories are listed.
func New(namespace, token string) *Client {
	return NewWithBaseURL(namespace, token, apiBaseURL)
}

func NewWithBaseURL(namespace, token, baseURL string) *Client {
	return &Client{
		client: resty.New().
			SetBaseURL(strings.TrimRight(baseURL, "/")).
			SetDisableWarn(true),
		namespace: namespace,
		token:     token,
	}
}

// ListRepositories returns the image repositories of the configured namespace
// as namespace/name. Quay does not expose a registry catalog endpoint, so this
// uses the public Quay API.
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	repos := make([]Repository, 0)

	for page := 1; ; page++ {
		var result listResponse

		resp, err := c.listRequest(ctx).
			SetResult(&result).
			SetQueryParam("limit", fmt.Sprint(pageSize)).
			SetQueryParam("page", fmt.Sprint(page)).
			Get("/api/v1/repository")
		if err != nil {
			return nil, fmt.Errorf("request.Get: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("quay api returned %s", resp.Status())
		}

		resp.Body.Close()

		for _, r := range result.Repositories {
			if r.Name == "" || (r.Kind != "" && r.Kind != "image") {
				continue
			}

			repos = append(repos, Repository{
				Name:      r.Name,
				Namespace: r.Namespace,
				IsPrivate: !r.IsPublic,
			})
		}

		if !result.HasAdditional {
			break
		}
	}

	return repos, nil
}

// Validate verifies that the configured namespace exists and that the token is
// accepted. It stops after the first page.
func (c *Client) Validate(ctx context.Context) error {
	resp, err := c.listRequest(ctx).
		SetQueryParam("limit", "1").
		Get("/api/v1/repository")
	if err != nil {
		return fmt.Errorf("request.Get: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("quay api returned %s", resp.Status())
	}

	return nil
}

func (c *Client) newRequest(ctx context.Context) *resty.Request {
	req := c.client.R().SetContext(ctx)

	if c.token != "" {
		req.SetHeader("Authorization", "Bearer "+c.token)
	}

	return req
}

// listRequest scopes the request to the namespace. Quay only returns public
// repositories anonymously when the public filter is set.
func (c *Client) listRequest(ctx context.Context) *resty.Request {
	req := c.newRequest(ctx).SetQueryParam("namespace", c.namespace)

	if c.token == "" {
		req.SetQueryParam("public", "true")
	}

	return req
}
