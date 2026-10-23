package quayclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListRepositories(t *testing.T) {
	var receivedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")

		if r.URL.Path != "/api/v1/repository" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		require.Equal(t, "octocat", r.URL.Query().Get("namespace"))

		w.Header().Set("Content-Type", "application/json")

		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `{"repositories":[{"namespace":"octocat","name":"three","kind":"image","is_public":false}],"has_additional":false}`)
			return
		}

		fmt.Fprint(w, `{"repositories":[{"namespace":"octocat","name":"one","kind":"image","is_public":true},{"namespace":"octocat","name":"two","kind":"image","is_public":false},{"namespace":"octocat","name":"chart","kind":"chart","is_public":true}],"has_additional":true}`)
	}))
	t.Cleanup(server.Close)

	client := NewWithBaseURL("octocat", "token123", server.URL)

	repos, err := client.ListRepositories(context.Background())
	require.NoError(t, err)
	require.Equal(t, []Repository{
		{Name: "one", Namespace: "octocat"},
		{Name: "two", Namespace: "octocat", IsPrivate: true},
		{Name: "three", Namespace: "octocat", IsPrivate: true},
	}, repos)
	require.Equal(t, "Bearer token123", receivedAuthHeader)
}

func TestListRepositoriesUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	client := NewWithBaseURL("octocat", "wrong", server.URL)

	_, err := client.ListRepositories(context.Background())
	require.ErrorContains(t, err, "401")
}

func TestListRepositoriesAnonymousUsesPublicFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "true", r.URL.Query().Get("public"))
		require.Empty(t, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"repositories":[{"namespace":"prometheus","name":"prometheus","kind":"image","is_public":true}],"has_additional":false}`)
	}))
	t.Cleanup(server.Close)

	client := NewWithBaseURL("prometheus", "", server.URL)

	repos, err := client.ListRepositories(context.Background())
	require.NoError(t, err)
	require.Equal(t, []Repository{{Name: "prometheus", Namespace: "prometheus"}}, repos)
}
