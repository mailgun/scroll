package scroll_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mailgun/holster/etcdutil"
	"github.com/mailgun/metrics"
	"github.com/mailgun/scroll"
	"github.com/mailgun/scroll/vulcand"
)

const (
	APPNAME = "example"
)

func Example() {
	// These environment variables provided by the environment,
	// we set them here to only to illustrate how `NewEtcdConfig()`
	// uses the environment to create a new etcd config
	os.Setenv("ETCD3_USER", "root")
	os.Setenv("ETCD3_PASSWORD", "rootpw")
	os.Setenv("ETCD3_ENDPOINT", "https://localhost:2379")
	// Set this to force connecting with TLS, but without cert verification
	os.Setenv("ETCD3_SKIP_VERIFY", "true")

	// If this is set to anything but empty string "", scroll will attempt
	// to retrieve the applications config from '/mailgun/configs/{env}/APPNAME'
	// and fill in the PublicAPI, ProtectedAPI, etc.. fields from that config
	os.Setenv("MG_ENV", "")

	// Create a new etc config from available environment variables
	cfg, err := etcdutil.NewEtcdConfig(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "while creating etcd config: %s\n", err)
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "while obtaining hostname: %s\n", err)
		return
	}

	// Send metrics to statsd @ localhost
	mc, err := metrics.NewWithOptions("localhost:8125",
		fmt.Sprintf("%s.%v", APPNAME, strings.Replace(hostname, ".", "_", -1)),
		metrics.Options{UseBuffering: true, FlushPeriod: time.Second})
	if err != nil {
		fmt.Fprintf(os.Stderr, "while initializing metrics: %s\n", err)
		return
	}

	app, err := scroll.NewAppWithConfig(scroll.AppConfig{
		Vulcand:          &vulcand.Config{Etcd: cfg},
		PublicAPIURL:     "http://api.mailgun.net:12121",
		ProtectedAPIURL:  "http://localhost:12121",
		PublicAPIHost:    "api.mailgun.net",
		ProtectedAPIHost: "localhost",
		Name:             APPNAME,
		ListenPort:       12121,
		Client:           mc,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "while initializing scroll: %s\n", err)
		return
	}

	app.AddHandler(
		scroll.Spec{
			Methods: []string{"GET"},
			Paths:   []string{"/hello"},
			Handler: func(w http.ResponseWriter, r *http.Request, params map[string]string) (interface{}, error) {
				return scroll.Response{"message": "Hello World"}, nil
			},
		},
	)

	// Start serving requests
	go func() {
		if err := app.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "while serving requests: %s\n", err)
		}
	}()

	// Wait for the endpoint to begin accepting connections
	waitFor("http://localhost:12121")

	// Get the response
	r, err := http.Get("http://localhost:12121/hello")
	if err != nil {
		fmt.Fprintf(os.Stderr, "GET request failed with: %s\n", err)
		return
	}
	content, _ := ioutil.ReadAll(r.Body)

	fmt.Println(string(content))

	// Shutdown the server and de-register routes with vulcand
	app.Stop()

	// Output:
	// {"message":"Hello World"}
}

// Waits for the endpoint to start accepting connections
func waitFor(url string) {
	after := time.After(time.Second * 10)
	_, err := http.Get(url)
	for err != nil {
		_, err = http.Get(url)
		select {
		case <-after:
			fmt.Fprintf(os.Stderr, "endpoint timeout: %s\n", err)
			os.Exit(1)
		default:
		}
	}
}
