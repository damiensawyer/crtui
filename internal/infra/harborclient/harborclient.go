package harborclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"resty.dev/v3"
)

const pageSize = 100

type Client struct {
	client *resty.Client
}

// Repository represents a Harbor repository.
type Repository struct {
	Name       string
	Visibility string
}

type harborProject struct {
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

type harborRepository struct {
	Name string `json:"name"`
}

// New returns a client for the Harbor API of the given registry URL. Harbor
// requires authentication to list projects, so anonymous access is not useful.
func New(baseURL, username, password string) *Client {
	client := resty.New().
		SetBaseURL(strings.TrimRight(baseURL, "/")).
		SetDisableWarn(true)

	if username != "" && password != "" {
		client = client.SetBasicAuth(username, password)
	}

	return &Client{client: client}
}

// ListRepositories returns every repository of every project as Harbor
// repository names (project/repository). Harbor's /v2/_catalog endpoint is
// only available to system admins, so the project API is used instead.
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	projects, err := c.listProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	repositories := make([]Repository, 0)

	for _, project := range projects {
		visibility := "private"
		if project.Metadata["public"] == "true" {
			visibility = "public"
		}

		repos, err := c.listProjectRepositories(ctx, project.Name)
		if err != nil {
			return nil, fmt.Errorf("list repositories of %q: %w", project.Name, err)
		}

		for _, repo := range repos {
			repositories = append(repositories, Repository{
				Name:       repo.Name,
				Visibility: visibility,
			})
		}
	}

	return repositories, nil
}

func (c *Client) listProjects(ctx context.Context) ([]harborProject, error) {
	projects := make([]harborProject, 0)

	for page := 1; ; page++ {
		var result []harborProject

		resp, err := c.client.R().SetContext(ctx).
			SetResult(&result).
			SetQueryParam("page", fmt.Sprint(page)).
			SetQueryParam("page_size", fmt.Sprint(pageSize)).
			Get("/api/v2.0/projects")
		if err != nil {
			return nil, fmt.Errorf("request.Get: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("harbor api returned %s", resp.Status())
		}

		resp.Body.Close()

		projects = append(projects, result...)

		if len(result) < pageSize {
			break
		}
	}

	return projects, nil
}

func (c *Client) listProjectRepositories(ctx context.Context, project string) ([]harborRepository, error) {
	repositories := make([]harborRepository, 0)

	for page := 1; ; page++ {
		var result []harborRepository

		resp, err := c.client.R().SetContext(ctx).
			SetResult(&result).
			SetQueryParam("page", fmt.Sprint(page)).
			SetQueryParam("page_size", fmt.Sprint(pageSize)).
			Get(fmt.Sprintf("/api/v2.0/projects/%s/repositories", url.PathEscape(project)))
		if err != nil {
			return nil, fmt.Errorf("request.Get: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("harbor api returned %s", resp.Status())
		}

		resp.Body.Close()

		repositories = append(repositories, result...)

		if len(result) < pageSize {
			break
		}
	}

	return repositories, nil
}

// Validate verifies that the credentials can list Harbor projects.
func (c *Client) Validate(ctx context.Context) error {
	resp, err := c.client.R().SetContext(ctx).
		SetQueryParam("page_size", "1").
		Get("/api/v2.0/projects")
	if err != nil {
		return fmt.Errorf("request.Get: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("harbor api returned %s", resp.Status())
	}

	return nil
}
