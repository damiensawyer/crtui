package harborclient

import (
	"context"
	"encoding/base64"
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

		switch r.URL.Path {
		case "/api/v2.0/projects":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"name":"library","metadata":{"public":"true"}},{"name":"team","metadata":{"public":"false"}}]`)
			return
		case "/api/v2.0/projects/library/repositories":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"name":"library/nginx"},{"name":"library/redis"}]`)
			return
		case "/api/v2.0/projects/team/repositories":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"name":"team/api"}]`)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := New(server.URL, "admin", "hunter2")

	repos, err := client.ListRepositories(context.Background())
	require.NoError(t, err)
	require.Equal(t, []Repository{
		{Name: "library/nginx", Visibility: "public"},
		{Name: "library/redis", Visibility: "public"},
		{Name: "team/api", Visibility: "private"},
	}, repos)
	require.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:hunter2")), receivedAuthHeader)
}

func TestListRepositoriesUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	client := New(server.URL, "admin", "wrong")

	_, err := client.ListRepositories(context.Background())
	require.ErrorContains(t, err, "401")
}
