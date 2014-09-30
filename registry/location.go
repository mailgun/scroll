package registry

import (
	"fmt"
	"strings"
)

type Location struct {
	ID       string
	Path     string
	Upstream string
	Scope    []Scope
}

func NewLocation(methods []string, path, upstream string, scope []Scope) *Location {
	path = convertPath(path)

	// if scope is not provided, assume public
	if len(scope) == 0 {
		scope = []Scope{ScopePublic}
	}

	return &Location{
		ID:       makeLocationID(methods, path),
		Path:     makeLocationPath(methods, path),
		Upstream: upstream,
		Scope:    scope,
	}
}

func (l *Location) String() string {
	return fmt.Sprintf("location(ID=%v, Path=%v, Upstream=%v, Scope=%v)", l.ID, l.Path, l.Upstream, l.Scope)
}

func makeLocationID(methods []string, path string) string {
	return strings.ToLower(strings.Replace(fmt.Sprintf("%v%v", strings.Join(methods, "."), path), "/", ".", -1))
}

func makeLocationPath(methods []string, path string) string {
	return fmt.Sprintf(`TrieRoute("%v", "%v")`, strings.Join(methods, `", "`), path)
}

// Convert router path to the format understood by vulcand.
//
// Effectively, just replaces curly brackets with angle brackets.
func convertPath(path string) string {
	return strings.Replace(strings.Replace(path, "{", "<", -1), "}", ">", -1)
}
