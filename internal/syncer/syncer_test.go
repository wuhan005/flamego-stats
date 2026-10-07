package syncer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoModCandidates(t *testing.T) {
	paths := []string{"go.mod", "vendor/github.com/foo/bar/go.mod", "api/go.mod", "go/beego/go.mod", "go/chi/go.mod", "go/flamego/go.mod", "internal/testdata/go.mod"}
	require.Equal(t,
		[]string{"go/flamego/go.mod", "api/go.mod", "go/beego/go.mod", "go/chi/go.mod"},
		goModCandidates(paths, "flamego"),
	)

	var many []string
	for i := 0; i < maxGoModFiles*2; i++ {
		many = append(many, fmt.Sprintf("go/f%02d/go.mod", i))
	}
	many = append(many, "go/Flamego-v1/go.mod")
	candidates := goModCandidates(many, "flamego")
	require.Len(t, candidates, maxGoModFiles)
	require.Equal(t, "go/Flamego-v1/go.mod", candidates[0])
}
