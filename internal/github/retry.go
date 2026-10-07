package github

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v92/github"
	"github.com/sirupsen/logrus"
)

const (
	maxRetries = 3
	// maxRateLimitWait is the longest wait for the primary rate limit to reset, the error is returned beyond it.
	maxRateLimitWait = time.Hour
)

// retry calls fn, and retries after a wait on rate limits, 5xx responses and network timeouts.
func retry[T any](ctx context.Context, fn func() (T, *github.Response, error)) (T, error) {
	for attempt := 0; ; attempt++ {
		v, _, err := fn()
		if err == nil {
			return v, nil
		}
		wait, ok := retryAfter(err, attempt)
		if !ok || attempt >= maxRetries {
			return v, err
		}

		logrus.WithContext(ctx).WithError(err).WithField("wait", wait.String()).Warn("GitHub API request failed, retrying")
		if err := sleep(ctx, wait); err != nil {
			return v, err
		}
	}
}

// retryAfter returns how long to wait before retrying, and whether err is retryable.
func retryAfter(err error, attempt int) (time.Duration, bool) {
	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		wait := time.Until(rateLimitErr.Rate.Reset.Time) + time.Second
		return max(wait, time.Second), wait <= maxRateLimitWait
	}

	var abuseErr *github.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		if abuseErr.RetryAfter != nil {
			return *abuseErr.RetryAfter, true
		}
		return time.Minute, true
	}

	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		return backoff(attempt), errResp.Response.StatusCode >= http.StatusInternalServerError
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return backoff(attempt), true
	}
	return 0, false
}

func backoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * 2 * time.Second
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
