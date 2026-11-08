package github

import (
	"context"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v92/github"
)

// maxPages caps the pages of a list, so a huge repository cannot drain the API quota.
const maxPages = 10

// Actor is the GitHub account behind an activity.
type Actor struct {
	// Login is the name in the commit if the commit author is not linked to any account, AvatarURL is empty then.
	Login     string
	AvatarURL string
	// Bot is true for the GitHub Apps, e.g. dependabot[bot].
	Bot bool
}

func newActor(user *github.User) Actor {
	login := user.GetLogin()
	return Actor{
		Login:     login,
		AvatarURL: user.GetAvatarURL(),
		Bot:       user.GetType() == "Bot" || strings.HasSuffix(login, "[bot]"),
	}
}

// ListOrgRepositories returns the public repositories of the organization.
func (c *Client) ListOrgRepositories(ctx context.Context, org string) ([]*Repository, error) {
	opts := &github.RepositoryListByOrgOptions{Type: "public", ListOptions: github.ListOptions{PerPage: 100}}
	repos, err := paginate(ctx, func(page int) ([]*github.Repository, *github.Response, error) {
		opts.Page = page
		return c.api.Repositories.ListByOrg(ctx, org, opts)
	}, nil)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	items := make([]*Repository, 0, len(repos))
	for _, repo := range repos {
		items = append(items, newRepository(repo))
	}
	return items, nil
}

// Commit is a commit on a branch.
type Commit struct {
	SHA     string
	Message string
	HTMLURL string
	Author  Actor
	// CommittedAt is when the commit landed on the branch, e.g. when the pull request was squash-merged.
	CommittedAt time.Time
	// Merge means the commit has more than one parent.
	Merge bool
}

// ListCommits returns the commits of the branch committed since the time, the newest first.
// It returns ErrNotFound if the repository is empty.
func (c *Client) ListCommits(ctx context.Context, fullName, branch string, since time.Time) ([]*Commit, error) {
	owner, name, err := splitFullName(fullName)
	if err != nil {
		return nil, err
	}
	opts := &github.CommitsListOptions{SHA: branch, Since: since, ListOptions: github.ListOptions{PerPage: 100}}
	commits, err := paginate(ctx, func(page int) ([]*github.RepositoryCommit, *github.Response, error) {
		opts.Page = page
		return c.api.Repositories.ListCommits(ctx, owner, name, opts)
	}, nil)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	items := make([]*Commit, 0, len(commits))
	for _, commit := range commits {
		author := newActor(commit.GetAuthor())
		if author.Login == "" {
			author = Actor{
				Login: commit.GetCommit().GetAuthor().GetName(),
				Bot:   strings.HasSuffix(commit.GetCommit().GetAuthor().GetName(), "[bot]"),
			}
		}
		items = append(items, &Commit{
			SHA:         commit.GetSHA(),
			Message:     commit.GetCommit().GetMessage(),
			HTMLURL:     commit.GetHTMLURL(),
			Author:      author,
			CommittedAt: commit.GetCommit().GetCommitter().GetDate().Time,
			Merge:       len(commit.Parents) > 1,
		})
	}
	return items, nil
}

// Issue is an issue or a pull request, GitHub lists both as the issues.
type Issue struct {
	ID          int64
	Number      int
	Title       string
	HTMLURL     string
	Author      Actor
	PullRequest bool
	Draft       bool
	CreatedAt   time.Time
	// ClosedAt is nil if the issue is open, MergedAt is nil if the pull request is not merged.
	ClosedAt *time.Time
	MergedAt *time.Time
}

// ListIssues returns the issues and the pull requests updated since the time, the recently updated first.
func (c *Client) ListIssues(ctx context.Context, fullName string, since time.Time) ([]*Issue, error) {
	owner, name, err := splitFullName(fullName)
	if err != nil {
		return nil, err
	}
	opts := &github.IssueListByRepoOptions{
		State:       "all",
		Sort:        "updated",
		Direction:   "desc",
		Since:       since,
		ListOptions: github.ListOptions{PerPage: 100},
	}
	issues, err := paginate(ctx, func(page int) ([]*github.Issue, *github.Response, error) {
		opts.ListOptions.Page = page
		return c.api.Issues.ListByRepo(ctx, owner, name, opts)
	}, nil)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	items := make([]*Issue, 0, len(issues))
	for _, issue := range issues {
		item := &Issue{
			ID:          issue.GetID(),
			Number:      issue.GetNumber(),
			Title:       issue.GetTitle(),
			HTMLURL:     issue.GetHTMLURL(),
			Author:      newActor(issue.GetUser()),
			PullRequest: issue.PullRequestLinks != nil,
			Draft:       issue.GetDraft(),
			CreatedAt:   issue.GetCreatedAt().Time,
			ClosedAt:    timePtr(issue.ClosedAt),
		}
		if issue.PullRequestLinks != nil {
			item.MergedAt = timePtr(issue.PullRequestLinks.MergedAt)
		}
		items = append(items, item)
	}
	return items, nil
}

// Release is a published release.
type Release struct {
	ID          int64
	Tag         string
	Name        string
	HTMLURL     string
	Author      Actor
	Prerelease  bool
	PublishedAt time.Time
}

// ListReleases returns the releases published since the time, the newest first.
func (c *Client) ListReleases(ctx context.Context, fullName string, since time.Time) ([]*Release, error) {
	owner, name, err := splitFullName(fullName)
	if err != nil {
		return nil, err
	}
	opts := &github.ListOptions{PerPage: 30}
	// The releases are ordered by the creation time, so the pages after an older one are not needed.
	releases, err := paginate(ctx, func(page int) ([]*github.RepositoryRelease, *github.Response, error) {
		opts.Page = page
		return c.api.Repositories.ListReleases(ctx, owner, name, opts)
	}, func(page []*github.RepositoryRelease) bool {
		return len(page) > 0 && page[len(page)-1].CreatedAt.Before(since)
	})
	if err != nil {
		return nil, wrapNotFound(err)
	}

	var items []*Release
	for _, release := range releases {
		if release.Draft || release.PublishedAt == nil || release.PublishedAt.Before(since) {
			continue
		}
		items = append(items, &Release{
			ID:          release.ID,
			Tag:         release.TagName,
			Name:        release.GetName(),
			HTMLURL:     release.HTMLURL,
			Author:      newActor(release.Author),
			Prerelease:  release.Prerelease,
			PublishedAt: release.PublishedAt.Time,
		})
	}
	return items, nil
}

// paginate calls list from the first page until the last one, maxPages, or done reports the following pages are not needed.
func paginate[T any](ctx context.Context, list func(page int) ([]T, *github.Response, error), done func(page []T) bool) ([]T, error) {
	var items []T
	for page, n := 1, 0; page != 0 && n < maxPages; n++ {
		next := 0
		got, err := retry(ctx, func() ([]T, *github.Response, error) {
			got, resp, err := list(page)
			if resp != nil {
				next = resp.NextPage
			}
			return got, resp, err
		})
		if err != nil {
			return nil, errors.Wrapf(err, "page %d", page)
		}
		items = append(items, got...)
		if done != nil && done(got) {
			break
		}
		page = next
	}
	return items, nil
}

func timePtr(t *github.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	return &t.Time
}
