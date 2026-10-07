package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// dependentsHTML renders a page in the layout of the GitHub dependents page, there is no next page if next is empty.
func dependentsHTML(total string, repos []string, next string) string {
	var b strings.Builder
	b.WriteString(`<html><body><div class="Box"><div class="Box-header"><div class="table-list-header-toggle states">`)
	fmt.Fprintf(&b, `<a class="btn-link selected" href="#"><svg></svg>
                %s
                Repositories
</a><a class="btn-link" href="#">91 Packages</a></div></div>`, total)
	for _, repo := range repos {
		owner, name, _ := strings.Cut(repo, "/")
		fmt.Fprintf(&b, `<div class="Box-row d-flex flex-items-center" data-test-id="dg-repo-pkg-dependent">
    <span class="f5 color-fg-muted">
      <a data-hovercard-type="user" href="/%s">%s</a> /
      <a class="text-bold" data-hovercard-type="repository" href="/%s">%s</a>
    </span></div>`, owner, owner, repo, name)
	}
	b.WriteString(`<div class="paginate-container"><div class="BtnGroup"><button class="btn BtnGroup-item" disabled="disabled">Previous</button>`)
	if next != "" {
		fmt.Fprintf(&b, `<a rel="nofollow" class="btn BtnGroup-item" href="%s">Next</a>`, next)
	} else {
		b.WriteString(`<button class="btn BtnGroup-item" disabled="disabled">Next</button>`)
	}
	b.WriteString(`</div></div></div></body></html>`)
	return b.String()
}

func TestListDependents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/flamego/flamego/network/dependents", r.URL.Path)
		require.Equal(t, "REPOSITORY", r.URL.Query().Get("dependent_type"))
		switch r.URL.Query().Get("dependents_after") {
		case "":
			_, _ = fmt.Fprint(w, dependentsHTML("1,203", []string{"gogs/gogs", "asoul-sig/asouldocs"},
				"/flamego/flamego/network/dependents?dependent_type=REPOSITORY&dependents_after=abc"))
		case "abc":
			_, _ = fmt.Fprint(w, dependentsHTML("1,203", []string{"Gogs/Gogs", "wuhan005/Elaina"}, ""))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient("")
	require.NoError(t, err)
	client.webURL = server.URL

	dependents, err := client.ListDependents(context.Background(), "flamego/flamego", DependentsOptions{})
	require.NoError(t, err)
	require.Equal(t, &Dependents{
		Total:        1203,
		Repositories: []string{"gogs/gogs", "asoul-sig/asouldocs", "wuhan005/Elaina"},
	}, dependents)
}

func TestListDependentsLayoutChanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, dependentsHTML("12", nil, ""))
	}))
	defer server.Close()

	client, err := NewClient("")
	require.NoError(t, err)
	client.webURL = server.URL

	_, err = client.ListDependents(context.Background(), "flamego/flamego", DependentsOptions{})
	require.ErrorContains(t, err, "page layout may have changed")
}
