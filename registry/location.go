package registry

import (
	"fmt"
	"strings"
)

type Location struct {
	ID       string
	APIHost  string
	Path     string
	Upstream string
}

func NewLocation(apiHost string, methods []string, path, upstream string) *Location {
	path = convertPath(path)
	return &Location{
		ID:       makeLocationID(methods, path),
		APIHost:  apiHost,
		Path:     makeLocationPath(methods, path),
		Upstream: upstream,
	}
}

func (l *Location) String() string {
	return fmt.Sprintf("location(ID=%v, APIHost=%v, Path=%v, Upstream=%v)", l.ID, l.APIHost, l.Path, l.Upstream)
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
