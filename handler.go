package scroll

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mailgun/gotools-log"
)

type Response map[string]interface{}

type HandlerConfig struct {
	Methods    []string
	Path       string
	Headers    []string
	MetricName string
	Register   bool
}

type HandlerFunc func(http.ResponseWriter, *http.Request, map[string]string) (interface{}, error)

func MakeHandler(app *App, fn HandlerFunc, config *HandlerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(0); err != nil {
			ReplyInternalError(w, fmt.Sprintf("Failed to parse request form: %v", err))
			return
		}

		start := time.Now()
		response, err := fn(w, r, mux.Vars(r))
		elapsedTime := time.Since(start)

		var status int
		if err != nil {
			response, status = responseAndStatusFor(err)
		} else {
			status = http.StatusOK
		}

		log.Infof("Request completed: status [%v] method [%v] path [%v] form [%v] time [%v] error [%v]",
			status, r.Method, r.URL, r.Form, elapsedTime, err)

		app.Stats.TrackRequest(config.MetricName, status, elapsedTime)

		Reply(w, response, status)
	}
}

type HandlerWithBodyFunc func(http.ResponseWriter, *http.Request, map[string]string, []byte) (interface{}, error)

func MakeHandlerWithBody(app *App, fn HandlerWithBodyFunc, config *HandlerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(0); err != nil {
			ReplyInternalError(w, fmt.Sprintf("Failed to parse request form: %v", err))
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			ReplyInternalError(w, fmt.Sprintf("Failed to read request body: %v", err))
			return
		}

		start := time.Now()
		response, err := fn(w, r, mux.Vars(r), body)
		elapsedTime := time.Since(start)

		var status int
		if err != nil {
			response, status = responseAndStatusFor(err)
		} else {
			status = http.StatusOK
		}

		log.Infof("Request completed: status [%v] method [%v] path [%v] form [%v] time [%v] error [%v]",
			status, r.Method, r.URL, r.Form, elapsedTime, err)

		app.Stats.TrackRequest(config.MetricName, status, elapsedTime)

		Reply(w, response, status)
	}
}

// Reply with the provided HTTP response and status code.
//
// Response body must be JSON-marshallable, otherwise the response
// will be "Internal Server Error".
func Reply(w http.ResponseWriter, response interface{}, status int) {
	// marshal the body of the response
	marshalledResponse, err := json.Marshal(response)
	if err != nil {
		ReplyInternalError(w, fmt.Sprintf("Failed to marshal response: %v %v", response, err))
		return
	}

	// write JSON response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(marshalledResponse)
}

func ReplyInternalError(w http.ResponseWriter, message string) {
	log.Errorf("Internal server error: %v", message)
	Reply(w, Response{"message": message}, http.StatusInternalServerError)
}
