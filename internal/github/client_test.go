package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v92/github"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a client with the token sending all the API and web requests to handler.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	baseURL := server.URL + "/"
	client, err := newClient("test-token", github.WithURLs(&baseURL, &baseURL))
	require.NoError(t, err)
	client.webURL = server.URL
	return client
}

func TestGetRepositoryVisibility(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/repos/acme/public":
			_, _ = fmt.Fprint(w, `{"id":1,"full_name":"acme/public","private":false,"visibility":"public"}`)
		case "/repos/acme/private":
			_, _ = fmt.Fprint(w, `{"id":2,"full_name":"acme/private","private":true,"visibility":"private"}`)
		case "/repos/acme/internal":
			_, _ = fmt.Fprint(w, `{"id":3,"full_name":"acme/internal","private":false,"visibility":"internal"}`)
		default:
			http.NotFound(w, r)
		}
	}))

	for fullName, public := range map[string]bool{"acme/public": true, "acme/private": false, "acme/internal": false} {
		repo, err := client.GetRepository(context.Background(), fullName)
		require.NoError(t, err)
		require.Equal(t, public, repo.Public, fullName)
	}
}

func TestSearchCodeRepositoriesExcludesPrivate(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"total_count":3,"items":[
			{"repository":{"full_name":"acme/public","private":false}},
			{"repository":{"full_name":"acme/secret","private":true}},
			{"repository":{"full_name":"acme/public","private":false}}
		]}`)
	}))

	fullNames, err := client.SearchCodeRepositories(context.Background(), `"github.com/flamego/flamego" filename:go.mod`)
	require.NoError(t, err)
	require.Equal(t, []string{"acme/public"}, fullNames)
}

func TestListDependentsIsAnonymous(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("Authorization"), "the dependents page must not see the private repositories of the token")
		_, _ = fmt.Fprint(w, dependentsHTML("1", []string{"acme/public"}, ""))
	}))

	dependents, err := client.ListDependents(context.Background(), "flamego/flamego", DependentsOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"acme/public"}, dependents.Repositories)
}
