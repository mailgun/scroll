package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/mailgun/scroll"
)

var host string
var port int

func init() {
	flag.StringVar(&host, "host", "0.0.0.0", "")
	flag.IntVar(&port, "port", 9000, "")
	flag.Parse()
}

func main() {
	name := "loadbalancer"

	appConfig := scroll.AppConfig{
		Name:             name,
		ListenIP:         host,
		ListenPort:       port,
		PublicAPIHost:    "public.local",
		ProtectedAPIHost: "private.local",
	}

	handlerSpec := scroll.Spec{
		Scope:   scroll.ScopePublic,
		Methods: []string{"GET"},
		Paths:   []string{"/"},
		Handler: index,
	}

	fmt.Printf("Starting %s on %s:%d...\n", name, host, port)

	app, err := scroll.NewAppWithConfig(appConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	app.AddHandler(handlerSpec)
	if err = app.Run(); err != nil {
		app.Stop() // Immediately un-register from vulcand
		os.Exit(1)
	}
}

func index(w http.ResponseWriter, r *http.Request, params map[string]string) (interface{}, error) {
	return scroll.Response{"message": "Hello World"}, nil
}
