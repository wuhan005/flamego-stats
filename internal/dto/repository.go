package dto

import (
	"time"

	"github.com/wuhan005/flamego-stats/internal/db"
)

// Repository is a GitHub repository depending on the module.
type Repository struct {
	FullName    string     `json:"fullName"`
	Name        string     `json:"name"`
	Owner       OwnerBrief `json:"owner"`
	Description string     `json:"description"`
	Homepage    string     `json:"homepage"`
	HTMLURL     string     `json:"htmlURL"`
	Language    string     `json:"language"`
	License     string     `json:"license"`
	Topics      []string   `json:"topics"`
	Fork        bool       `json:"fork"`
	Archived    bool       `json:"archived"`
	Stars       int        `json:"stars"`
	Forks       int        `json:"forks"`
	Watchers    int        `json:"watchers"`
	OpenIssues  int        `json:"openIssues"`
	CreatedAt   time.Time  `json:"createdAt"`
	PushedAt    time.Time  `json:"pushedAt"`
	// GoModPath is the path of the go.mod requiring the module. If not found, it is empty along with GoVersion and ModuleVersion.
	GoModPath      string    `json:"goModPath"`
	GoVersion      string    `json:"goVersion"`
	ModuleVersion  string    `json:"moduleVersion"`
	RelatedModules []string  `json:"relatedModules"`
	SyncedAt       time.Time `json:"syncedAt"`
} // @name Repository

// OwnerBrief is the owner of a repository.
type OwnerBrief struct {
	Login     string `json:"login"`
	Type      string `json:"type" enums:"User,Organization"`
	AvatarURL string `json:"avatarURL"`
	HTMLURL   string `json:"htmlURL"`
} // @name OwnerBrief

func ToRepository(repository *db.Repository) *Repository {
	return &Repository{
		FullName: repository.FullName,
		Name:     repository.Name,
		Owner: OwnerBrief{
			Login:     repository.OwnerLogin,
			Type:      repository.OwnerType,
			AvatarURL: repository.OwnerAvatarURL,
			HTMLURL:   ownerHTMLURL(repository.OwnerLogin),
		},
		Description:    repository.Description,
		Homepage:       repository.Homepage,
		HTMLURL:        repository.HTMLURL,
		Language:       repository.Language,
		License:        repository.License,
		Topics:         nonNil(repository.Topics),
		Fork:           repository.Fork,
		Archived:       repository.Archived,
		Stars:          repository.Stars,
		Forks:          repository.Forks,
		Watchers:       repository.Watchers,
		OpenIssues:     repository.OpenIssues,
		CreatedAt:      repository.GitHubCreatedAt,
		PushedAt:       repository.PushedAt,
		GoModPath:      repository.GoModPath,
		GoVersion:      repository.GoVersion,
		ModuleVersion:  repository.ModuleVersion,
		RelatedModules: nonNil(repository.RelatedModules),
		SyncedAt:       repository.SyncedAt,
	}
}

// ListRepositoriesResp is the response of the repository list.
type ListRepositoriesResp struct {
	Repositories []*Repository `json:"repositories"`
	Total        int64         `json:"total"`
} // @name ListRepositoriesResp

func ownerHTMLURL(login string) string {
	return "https://github.com/" + login
}

// nonNil makes the empty list encoded as [] rather than null.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
