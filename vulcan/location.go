package vulcan

import (
	"fmt"
	"strings"

	"github.com/mailgun/scroll/vulcan/middleware"
)

const (
	defaultFailoverPredicate = "(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2"
)

type Location struct {
	ID          string
	Host        string
	Path        string
	Upstream    string
	Options     LocationOptions
	Middlewares []middleware.Middleware
}

type LocationOptions struct {
	FailoverPredicate string `json:"FailoverPredicate"`
}

func (o LocationOptions) String() string {
	return fmt.Sprintf("LocationOptions(FailoverPredicate=%v)", o.FailoverPredicate)
}

func NewLocation(host string, methods []string, path, upstream string, middlewares []middleware.Middleware) *Location {
	path = convertPath(path)

	return &Location{
		ID:       makeLocationID(methods, path),
		Host:     host,
		Path:     makeLocationPath(methods, path),
		Upstream: upstream,
		Options: LocationOptions{
			FailoverPredicate: defaultFailoverPredicate,
		},
		Middlewares: middlewares,
	}
}

func (l *Location) String() string {
	return fmt.Sprintf("Location(ID=%v, Host=%v, Path=%v, Upstream=%v, Options=%v, Middlewares=%v)",
		l.ID, l.Host, l.Path, l.Upstream, l.Options, l.Middlewares)
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
