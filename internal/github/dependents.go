package github

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
)

// DependentsOptions are the options of listing the dependents.
type DependentsOptions struct {
	// PackageID is the package filter of the dependents page, the default package of the repository is used if empty.
	PackageID string
	// PageDelay is the delay between the pages.
	PageDelay time.Duration
	// MaxPages stops the endless paging in case the page layout changes, it defaults to 1000 if not positive.
	MaxPages int
}

// Dependents is the repositories listed in the "Used by" page.
type Dependents struct {
	// Total is the number of dependents shown on the page, which GitHub notes is approximate, it is 0 if failed to parse.
	Total int
	// Repositories are the distinct owner/name in the order of the pages.
	Repositories []string
}

// ListDependents pages through the network/dependents page of the fullName repository to list all the repositories depending on it.
// Neither the GitHub REST nor the GraphQL API provides the "Used by" list, so the web page is parsed.
// The page is requested anonymously, so it never lists the private repositories accessible to the token.
func (c *Client) ListDependents(ctx context.Context, fullName string, opts DependentsOptions) (*Dependents, error) {
	if _, _, err := splitFullName(fullName); err != nil {
		return nil, err
	}
	if opts.MaxPages <= 0 {
		opts.MaxPages = 1000
	}

	query := url.Values{"dependent_type": {"REPOSITORY"}}
	if opts.PackageID != "" {
		query.Set("package_id", opts.PackageID)
	}
	next := c.webURL + "/" + fullName + "/network/dependents?" + query.Encode()

	dependents := &Dependents{}
	seen := make(map[string]struct{})
	for page := 0; next != ""; page++ {
		if page >= opts.MaxPages {
			return nil, errors.Newf("exceeded %d pages", opts.MaxPages)
		}
		if page > 0 {
			if err := sleep(ctx, opts.PageDelay); err != nil {
				return nil, err
			}
		}

		p, err := c.fetchDependentsPage(ctx, next)
		if err != nil {
			return nil, errors.Wrapf(err, "page %d", page+1)
		}
		if page == 0 {
			dependents.Total = p.Total
		}
		for _, repo := range p.Repositories {
			key := strings.ToLower(repo)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			dependents.Repositories = append(dependents.Repositories, repo)
		}
		next = p.Next
	}
	return dependents, nil
}

func (c *Client) fetchDependentsPage(ctx context.Context, pageURL string) (*dependentsPage, error) {
	for attempt := 0; ; attempt++ {
		page, wait, err := c.tryFetchDependentsPage(ctx, pageURL)
		if err == nil {
			return page, nil
		}
		if wait == 0 || attempt >= maxRetries {
			return nil, err
		}

		logrus.WithContext(ctx).WithError(err).WithField("wait", wait.String()).Warn("Failed to fetch dependents page, retrying")
		if err := sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

// tryFetchDependentsPage fetches a dependents page, the error is retryable after wait if wait is not 0.
func (c *Client) tryFetchDependentsPage(ctx context.Context, pageURL string) (_ *dependentsPage, wait time.Duration, _ error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, 0, errors.Wrap(err, "new request")
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, ctx.Err()
		}
		return nil, 5 * time.Second, errors.Wrap(err, "do request")
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		wait := time.Minute
		if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 {
			wait = time.Duration(seconds) * time.Second
		}
		return nil, wait, errors.New("rate limited")
	case resp.StatusCode >= http.StatusInternalServerError:
		return nil, 5 * time.Second, errors.Newf("unexpected status %d", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, 0, errors.Newf("unexpected status %d", resp.StatusCode)
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, 0, errors.Wrap(err, "parse page URL")
	}
	page, err := parseDependentsPage(resp.Body, base)
	if err != nil {
		return nil, 0, errors.Wrap(err, "parse page")
	}
	return page, 0, nil
}

type dependentsPage struct {
	Total        int
	Repositories []string
	// Next is the absolute URL of the next page, it is empty on the last page.
	Next string
}

var numberPattern = regexp.MustCompile(`[\d,]+`)

func parseDependentsPage(r io.Reader, base *url.URL) (*dependentsPage, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	page := &dependentsPage{}
	if total := numberPattern.FindString(doc.Find(".table-list-header-toggle a.selected").First().Text()); total != "" {
		page.Total, _ = strconv.Atoi(strings.ReplaceAll(total, ",", ""))
	}

	var parseErr error
	doc.Find(`[data-test-id="dg-repo-pkg-dependent"]`).EachWithBreak(func(_ int, row *goquery.Selection) bool {
		href, _ := row.Find(`a[data-hovercard-type="repository"]`).First().Attr("href")
		fullName := strings.Trim(href, "/")
		if _, _, err := splitFullName(fullName); err != nil {
			parseErr = errors.Newf("unexpected repository link %q", href)
			return false
		}
		page.Repositories = append(page.Repositories, fullName)
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	if page.Total > 0 && len(page.Repositories) == 0 {
		return nil, errors.New("no dependents found, the page layout may have changed")
	}

	doc.Find(".paginate-container a").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		if strings.TrimSpace(a.Text()) != "Next" {
			return true
		}
		href, _ := a.Attr("href")
		if next, err := base.Parse(href); err == nil && href != "" {
			page.Next = next.String()
		}
		return false
	})
	return page, nil
}
