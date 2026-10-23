package registryclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ksckaan1/crtui/internal/core/enums/registrytype"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	client := New("https://registry.example.com", "user", "pass")
	require.NotNil(t, client)
	require.Equal(t, "https://registry.example.com", client.BaseURL)
	require.Equal(t, "user", client.username)
	require.Equal(t, "pass", client.password)
	err := client.Close()
	require.NoError(t, err)
}

func TestNewWithoutCredentials(t *testing.T) {
	client := New("https://registry.example.com", "", "")
	require.NotNil(t, client)
	require.Equal(t, "https://registry.example.com", client.BaseURL)
	err := client.Close()
	require.NoError(t, err)
}

func TestSetTransportHTTP3(t *testing.T) {
	client := New("https://registry.example.com", "user", "pass")
	require.NotNil(t, client)
	require.False(t, client.supportsHTTP3)

	client.SetTransport(true)
	require.True(t, client.supportsHTTP3)

	client.SetTransport(false)
	require.False(t, client.supportsHTTP3)

	err := client.Close()
	require.NoError(t, err)
}

func TestSetTransportNoChange(t *testing.T) {
	client := New("https://registry.example.com", "user", "pass")
	require.NotNil(t, client)

	client.SetTransport(true)
	require.True(t, client.supportsHTTP3)

	client.SetTransport(true)
	require.True(t, client.supportsHTTP3)

	err := client.Close()
	require.NoError(t, err)
}

func TestGetRegistryInfoDetectsHarbor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
		case "/api/v2.0/ping":
			fmt.Fprint(w, "Pong")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := New(server.URL, "", "")
	t.Cleanup(func() { _ = client.Close() })

	info, err := client.GetRegistryInfo(context.Background())
	require.NoError(t, err)
	require.Equal(t, registrytype.Harbor, info.Type)
}

func TestGetRegistryInfoKeepsDockerType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := New(server.URL, "", "")
	t.Cleanup(func() { _ = client.Close() })

	info, err := client.GetRegistryInfo(context.Background())
	require.NoError(t, err)
	require.Equal(t, registrytype.Docker, info.Type)
}

func TestGetRegistryInfoDetectsZot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
		case "/v2/_oci/ext/discover":
			fmt.Fprint(w, `{"extensions":[{"name":"_zot","endpoints":["/v2/_zot/ext/search"]}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := New(server.URL, "", "")
	t.Cleanup(func() { _ = client.Close() })

	info, err := client.GetRegistryInfo(context.Background())
	require.NoError(t, err)
	require.Equal(t, registrytype.Zot, info.Type)
}
