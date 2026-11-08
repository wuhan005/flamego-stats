package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/dto"
)

func TestActivityCursor(t *testing.T) {
	want := &db.ActivityCursor{OccurredAt: time.Date(2026, 1, 2, 3, 4, 5, 6000, time.UTC), ID: 42}
	got, err := parseActivityCursor(formatActivityCursor(want))
	require.NoError(t, err)
	require.True(t, want.OccurredAt.Equal(got.OccurredAt))
	require.Equal(t, want.ID, got.ID)

	for _, invalid := range []string{"42", "x-1", "1-x"} {
		_, err := parseActivityCursor(invalid)
		require.Error(t, err, invalid)
	}
}

func TestFillDays(t *testing.T) {
	firstDay := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
	got := fillDays([]*db.DailyCount{{Day: "2026-03-01", Count: 5}}, firstDay, 3)
	require.Equal(t, []*dto.DailyCount{
		{Date: "2026-02-27", Count: 0},
		{Date: "2026-02-28", Count: 0},
		{Date: "2026-03-01", Count: 5},
	}, got)
}
