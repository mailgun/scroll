package testutils

import (
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/mailgun/scroll"
)

// TestApp wraps a regular app adding features that can be used in unit tests.
type TestApp struct {
	RestHelper
	app        *scroll.App
	testServer *httptest.Server
}

// NewTestApp creates a new app should be used in unit tests.
func NewTestApp() *TestApp {
	router := mux.NewRouter()
	config := scroll.AppConfig{
		Name:     "test",
		Router:   router}

	app, err := scroll.NewAppWithConfig(config)
	if err != nil {
		panic(err)
	}
	return &TestApp{
		RestHelper{},
		app,
		httptest.NewServer(router),
	}
}

// GetApp returns an underlying "real" app for the test app.
func (testApp *TestApp) GetApp() *scroll.App {
	return testApp.app
}

// GetURL returns the base URL of the underlying test server.
func (testApp *TestApp) GetURL() string {
	return testApp.testServer.URL
}

// Close shuts down the underlying test server.
func (testApp *TestApp) Close() {
	testApp.testServer.Close()
}
