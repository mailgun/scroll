**WORK IN PROGRESS**

scroll
======

Scroll is a lightweight library for building Go HTTP services at Mailgun.

Example
-------

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/mailgun/scroll"
)

func handler(w http.ResponseWriter, r *http.Request, params map[string]string) (interface{}, error) {
	return scroll.Response{
		"message": fmt.Sprintf("Resource ID: %v", params["resourceID"]),
	}, nil
}

func main() {
	// create an app
	appConfig := scroll.AppConfig{
		Name:       "scrollexample",
		ListenIP:   "0.0.0.0",
		ListenPort: 8080,
		Register:   false,
	}
	app := scroll.NewAppWithConfig(appConfig)

	// register a handler
	handlerSpec := scroll.Spec{

		Methods:  []string{"GET", "POST"},
		Path:     "/resources/{resourceID}",
		Register: false,
		Handler:  handler,
	}

	app.AddHandler(handlerSpec)

	// start the app
	app.Run()
}
```

Build info
----------

Scroll apps automatically provide an HTTP endpoint `/build_info` that displays information about the running binary, like when it was built, what commit it was built from, and a link to the Github view of that commit. To use this feature, pass the following flag to `go build` or `go install`.

    -ldflags "-X `go list -f '{{join .Deps "\n"}}' | grep 'mailgun/scroll$'`.build '`git log -1 --oneline`; `date`; `go list`'"

Example usage in a Makefile:

    all:
        go install -ldflags "-X `go list -f '{{join .Deps "\n"}}' | grep 'scroll$'`.build '`git log -1 --oneline`; `date`; `go list`'" github.com/mailgun/gatekeeper


Explanation of the flag: `go build`, `go install`, and several other `go` subcommands allow passing flags to [go tool ld](http://golang.org/cmd/ld/) through `-ldflags`. The one we are passing here is `-X`, which sets the value of an otherwise uninitialized string variable. The variable we are setting is `github.com/mailgun/scroll.build`, but since most Mailgun binaries use Godep, we might need to set `github.com/mailgun/<APP>/Godeps/_workspace/src/github.com/mailgun/scroll.build` instead. To handle either case, we programatically find the name of a transitive dependency ending in "mailgun/scroll" and set its `build` variabl. That's the ``go list -f '{{join .Deps "\n"}}' | grep 'mailgun/scroll$'`.build`` part of the flag. Finally, we use ``'`git log -1 --oneline`; `date`; `go list`'`` to capture the build information and semicolon-separate it for `scroll` to parse.

Note that this endpoint is not registered in vulcan.
