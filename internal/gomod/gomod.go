package gomod

import (
	"path"
	"slices"
	"strings"

	"github.com/cockroachdb/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

// Info is the dependency of the module in a go.mod file.
type Info struct {
	// GoVersion is the version of the go directive, e.g. 1.22.0.
	GoVersion string
	// Version is the required version of the module. It is the replacement version if replaced by another version
	// of the same module, and the required version if replaced by a local path.
	Version string
	// MinorVersion is the vMAJOR.MINOR of Version, it is empty if Version is invalid.
	MinorVersion string
	// RelatedModules are the other required modules with the same prefix (e.g. github.com/flamego/) in lexical order.
	RelatedModules []string
}

var ErrNotRequired = errors.New("module is not required")

// Parse parses the dependency on module in the go.mod, it returns ErrNotRequired if the module is not required.
func Parse(file string, data []byte, module string) (*Info, error) {
	f, err := modfile.Parse(file, data, nil)
	if err != nil {
		// The go.mod of other repositories may contain directives unknown to the current toolchain.
		f, err = modfile.ParseLax(file, data, nil)
		if err != nil {
			return nil, errors.Wrap(err, "parse")
		}
	}

	info := &Info{}
	if f.Go != nil {
		info.GoVersion = f.Go.Version
	}

	prefix := path.Dir(module) + "/"
	required := false
	for _, r := range f.Require {
		switch {
		case strings.EqualFold(r.Mod.Path, module):
			required = true
			info.Version = r.Mod.Version
		case strings.HasPrefix(strings.ToLower(r.Mod.Path), strings.ToLower(prefix)):
			info.RelatedModules = append(info.RelatedModules, r.Mod.Path)
		}
	}
	if !required {
		return nil, ErrNotRequired
	}

	for _, r := range f.Replace {
		if strings.EqualFold(r.Old.Path, module) && (r.Old.Version == "" || r.Old.Version == info.Version) && r.New.Version != "" {
			info.Version = r.New.Version
		}
	}

	info.MinorVersion = semver.MajorMinor(info.Version)
	slices.Sort(info.RelatedModules)
	info.RelatedModules = slices.Compact(info.RelatedModules)
	return info, nil
}
