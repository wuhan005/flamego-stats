package gomod

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const module = "github.com/flamego/flamego"

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		want *Info
	}{
		{
			name: "require",
			data: `module example.com/app

go 1.22.0

require (
	github.com/flamego/flamego v1.9.4
	github.com/flamego/session v1.6.0
	github.com/flamego/binding v1.3.0
	github.com/sirupsen/logrus v1.9.3
)
`,
			want: &Info{
				GoVersion:      "1.22.0",
				Version:        "v1.9.4",
				MinorVersion:   "v1.9",
				RelatedModules: []string{"github.com/flamego/binding", "github.com/flamego/session"},
			},
		},
		{
			name: "indirect pseudo-version",
			data: `module example.com/app

go 1.18

require github.com/flamego/flamego v0.0.0-20210515130931-267e7ab3ad1a // indirect
`,
			want: &Info{
				GoVersion:    "1.18",
				Version:      "v0.0.0-20210515130931-267e7ab3ad1a",
				MinorVersion: "v0.0",
			},
		},
		{
			name: "replace version",
			data: `module example.com/app

require github.com/flamego/flamego v1.8.0

replace github.com/flamego/flamego => github.com/flamego/flamego v1.9.0
`,
			want: &Info{Version: "v1.9.0", MinorVersion: "v1.9"},
		},
		{
			name: "replace local path",
			data: `module example.com/app

require github.com/flamego/flamego v1.8.0

replace github.com/flamego/flamego => ../flamego
`,
			want: &Info{Version: "v1.8.0", MinorVersion: "v1.8"},
		},
		{
			name: "unknown directive",
			data: `module example.com/app

go 1.30

unknowndirective foo

require github.com/flamego/flamego v1.9.4
`,
			want: &Info{GoVersion: "1.30", Version: "v1.9.4", MinorVersion: "v1.9"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse("go.mod", []byte(tc.data), module)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestParseNotRequired(t *testing.T) {
	_, err := Parse("go.mod", []byte(`module example.com/app

require github.com/flamego/session v1.6.0
`), module)
	require.ErrorIs(t, err, ErrNotRequired)
}
