package github

import (
	"context"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v92/github"
)

const userAgent = "flamego-stats (+https://github.com/wuhan005/flamego-stats)"

// ErrNotFound means the repository or the file does not exist, or the repository is empty.
var ErrNotFound = errors.New("not found")

// Client accesses the GitHub REST API and the dependents page, all the requests wait and retry on rate limits.
type Client struct {
	api  *github.Client
	http *http.Client
	// webURL is the base URL of the GitHub website without the trailing slash.
	webURL string
	// authenticated is required by the Code Search API.
	authenticated bool
}

type options struct {
	apiURL string
	webURL string
}

// Option configures the Client.
type Option func(*options)

// WithURLs sets the base URLs of the REST API (with the trailing slash) and the website, e.g. for GitHub Enterprise.
func WithURLs(apiURL, webURL string) Option {
	return func(o *options) {
		o.apiURL = apiURL
		o.webURL = strings.TrimSuffix(webURL, "/")
	}
}

// NewClient creates a GitHub client, the requests are anonymous if token is empty.
func NewClient(token string, opts ...Option) (*Client, error) {
	o := options{webURL: "https://github.com"}
	for _, opt := range opts {
		opt(&o)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	apiOpts := []github.ClientOptionsFunc{
		github.WithHTTPClient(httpClient),
		github.WithUserAgent(userAgent),
	}
	if o.apiURL != "" {
		apiOpts = append(apiOpts, github.WithURLs(&o.apiURL, &o.apiURL))
	}
	if token != "" {
		apiOpts = append(apiOpts, github.WithAuthToken(token))
	}
	api, err := github.NewClient(apiOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "new client")
	}
	return &Client{
		api:           api,
		http:          httpClient,
		webURL:        o.webURL,
		authenticated: token != "",
	}, nil
}

// Repository is the metadata of a GitHub repository.
type Repository struct {
	ID             int64
	OwnerID        int64
	OwnerLogin     string
	OwnerType      string
	OwnerAvatarURL string

	Name          string
	FullName      string
	Description   string
	Homepage      string
	HTMLURL       string
	Language      string
	License       string
	Topics        []string
	DefaultBranch string
	Fork          bool
	Archived      bool
	// Public is false for the private and the internal repositories, which the token may be able to access.
	Public bool

	Stars      int
	Forks      int
	Watchers   int
	OpenIssues int

	CreatedAt time.Time
	UpdatedAt time.Time
	PushedAt  time.Time
}

// GetRepository returns the owner/name repository, the renamed or transferred one is returned with its new name.
// It returns ErrNotFound if the repository does not exist or is not accessible.
func (c *Client) GetRepository(ctx context.Context, fullName string) (*Repository, error) {
	owner, name, err := splitFullName(fullName)
	if err != nil {
		return nil, err
	}
	repo, err := retry(ctx, func() (*github.Repository, *github.Response, error) {
		return c.api.Repositories.Get(ctx, owner, name)
	})
	if err != nil {
		return nil, wrapNotFound(err)
	}

	return &Repository{
		ID:             repo.GetID(),
		OwnerID:        repo.GetOwner().GetID(),
		OwnerLogin:     repo.GetOwner().GetLogin(),
		OwnerType:      repo.GetOwner().GetType(),
		OwnerAvatarURL: repo.GetOwner().GetAvatarURL(),
		Name:           repo.GetName(),
		FullName:       repo.GetFullName(),
		Description:    repo.GetDescription(),
		Homepage:       repo.GetHomepage(),
		HTMLURL:        repo.GetHTMLURL(),
		Language:       repo.GetLanguage(),
		License:        repo.GetLicense().GetSPDXID(),
		Topics:         repo.Topics,
		DefaultBranch:  repo.GetDefaultBranch(),
		Fork:           repo.GetFork(),
		Archived:       repo.GetArchived(),
		Public:         isPublic(repo),
		Stars:          repo.GetStargazersCount(),
		Forks:          repo.GetForksCount(),
		Watchers:       repo.GetSubscribersCount(),
		OpenIssues:     repo.GetOpenIssuesCount(),
		CreatedAt:      repo.GetCreatedAt().Time,
		UpdatedAt:      repo.GetUpdatedAt().Time,
		PushedAt:       repo.GetPushedAt().Time,
	}, nil
}

// GetFile returns the content of filePath at ref of the repository.
// It returns ErrNotFound if the file does not exist.
func (c *Client) GetFile(ctx context.Context, fullName, ref, filePath string) ([]byte, error) {
	owner, name, err := splitFullName(fullName)
	if err != nil {
		return nil, err
	}
	file, err := retry(ctx, func() (*github.RepositoryContent, *github.Response, error) {
		file, _, resp, err := c.api.Repositories.GetContents(ctx, owner, name, filePath, &github.RepositoryContentGetOptions{Ref: ref})
		return file, resp, err
	})
	if err != nil {
		return nil, wrapNotFound(err)
	}
	// file is nil if the path is a directory.
	if file == nil || file.GetType() != "file" {
		return nil, ErrNotFound
	}
	content, err := file.GetContent()
	if err != nil {
		return nil, errors.Wrap(err, "decode content")
	}
	return []byte(content), nil
}

// FindFiles returns the paths of the files named fileName in the recursive tree at ref, ordered by depth and then lexically.
// GitHub truncates the trees of huge repositories, only the listed part is searched then.
// It returns ErrNotFound if the repository is empty.
func (c *Client) FindFiles(ctx context.Context, fullName, ref, fileName string) ([]string, error) {
	owner, name, err := splitFullName(fullName)
	if err != nil {
		return nil, err
	}
	tree, err := retry(ctx, func() (*github.Tree, *github.Response, error) {
		return c.api.Git.GetTree(ctx, owner, name, ref, true)
	})
	if err != nil {
		return nil, wrapNotFound(err)
	}

	var paths []string
	for _, entry := range tree.Entries {
		if entry.GetType() == "blob" && path.Base(entry.GetPath()) == fileName {
			paths = append(paths, entry.GetPath())
		}
	}
	slices.SortFunc(paths, func(a, b string) int {
		if d := strings.Count(a, "/") - strings.Count(b, "/"); d != 0 {
			return d
		}
		return strings.Compare(a, b)
	})
	return paths, nil
}

// codeSearchLimit is the maximum number of results a Code Search API query can page through.
const codeSearchLimit = 1000

// CodeSearchRepository is a repository with the files matching the code search.
type CodeSearchRepository struct {
	FullName string
	// Paths are the paths of the matched files in the repository.
	Paths []string
}

// SearchCodeRepositories searches code by the Code Search API, and returns the matched repositories of the first 1000 results at most.
func (c *Client) SearchCodeRepositories(ctx context.Context, query string) ([]*CodeSearchRepository, error) {
	if !c.authenticated {
		return nil, errors.New("code search requires an authentication token")
	}

	byName := make(map[string]*CodeSearchRepository)
	var repos []*CodeSearchRepository
	opts := &github.SearchOptions{ListOptions: github.ListOptions{Page: 1, PerPage: 100}}
	for fetched := 0; fetched < codeSearchLimit; opts.Page++ {
		result, err := retry(ctx, func() (*github.CodeSearchResult, *github.Response, error) {
			return c.api.Search.Code(ctx, query, opts)
		})
		if err != nil {
			return nil, errors.Wrapf(err, "search page %d", opts.Page)
		}
		for _, item := range result.CodeResults {
			fullName := item.GetRepository().GetFullName()
			// The results include the private repositories accessible to the token.
			if !isPublic(item.GetRepository()) || fullName == "" {
				continue
			}
			key := strings.ToLower(fullName)
			repo, ok := byName[key]
			if !ok {
				repo = &CodeSearchRepository{FullName: fullName}
				byName[key] = repo
				repos = append(repos, repo)
			}
			if p := item.GetPath(); p != "" && !slices.Contains(repo.Paths, p) {
				repo.Paths = append(repo.Paths, p)
			}
		}

		fetched += len(result.CodeResults)
		if len(result.CodeResults) == 0 || fetched >= result.GetTotal() {
			break
		}
	}
	return repos, nil
}

// isPublic reports whether the repository is public, the visibility is absent in some responses like the code search.
func isPublic(repo *github.Repository) bool {
	if repo.GetPrivate() {
		return false
	}
	visibility := repo.GetVisibility()
	return visibility == "" || visibility == "public"
}

func splitFullName(fullName string) (owner, name string, err error) {
	owner, name, ok := strings.Cut(fullName, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return "", "", errors.Newf("invalid repository name %q", fullName)
	}
	return owner, name, nil
}

// wrapNotFound converts 404 and the empty repository (409) to ErrNotFound.
func wrapNotFound(err error) error {
	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		switch errResp.Response.StatusCode {
		case http.StatusNotFound, http.StatusConflict:
			return ErrNotFound
		}
	}
	return err
}
