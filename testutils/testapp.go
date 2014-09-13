package testutils

import (
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/mailgun/scroll"
)

// TestApp should be used in unit tests.
type TestApp struct {
	RestHelper
	*scroll.App
	testServer *httptest.Server
}

func NewTestApp() *TestApp {
	router := mux.NewRouter()

	config := &scroll.AppConfig{
		Name:     "testapp",
		Host:     "0.0.0.0",
		Port:     5060,
		Router:   router,
		Register: false,
	}

	testServer := httptest.NewServer(router)

	return &TestApp{RestHelper{}, scroll.NewApp(config), testServer}
}
