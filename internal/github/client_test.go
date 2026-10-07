package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// newTestClient creates a client with the token sending all the API and web requests to handler.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient("test-token", WithURLs(server.URL+"/", server.URL))
	require.NoError(t, err)
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
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "flamego filename:go.mod", r.URL.Query().Get("q"))
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = fmt.Fprint(w, `{"total_count":4,"items":[
				{"path":"go.mod","repository":{"full_name":"acme/public","private":false}},
				{"path":"go.mod","repository":{"full_name":"acme/secret","private":true}}
			]}`)
		case "2":
			_, _ = fmt.Fprint(w, `{"total_count":4,"items":[
				{"path":"api/go.mod","repository":{"full_name":"acme/public","private":false}},
				{"path":"go.mod","repository":{"full_name":"acme/other","private":false}}
			]}`)
		default:
			t.Fatalf("unexpected page %q", r.URL.Query().Get("page"))
		}
	}))

	repos, err := client.SearchCodeRepositories(context.Background(), "flamego filename:go.mod")
	require.NoError(t, err)
	require.Equal(t, []*CodeSearchRepository{
		{FullName: "acme/public", Paths: []string{"go.mod", "api/go.mod"}},
		{FullName: "acme/other", Paths: []string{"go.mod"}},
	}, repos)
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
