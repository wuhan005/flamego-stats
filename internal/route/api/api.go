package api

import (
	"github.com/wuhan005/flamego-stats/internal/context"
	"github.com/wuhan005/flamego-stats/internal/db"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// scopeOptions parses the includeForks query parameter shared by the APIs.
func scopeOptions(ctx context.Context) db.ScopeOptions {
	return db.ScopeOptions{IncludeForks: ctx.QueryBool("includeForks")}
}

// limit parses the limit query parameter, it defaults to 20 and is at most 100.
func limit(ctx context.Context) int {
	n := ctx.QueryInt("limit")
	if n <= 0 {
		return defaultLimit
	}
	return min(n, maxLimit)
}
