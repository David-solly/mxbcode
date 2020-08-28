package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/pkg/testutil/assert"
	"github.com/go-chi/chi"
)

var rt *chi.Mux

func TestStretchAPI(t *testing.T) {
	t.Run("TEST stretch api", func(t *testing.T) {
		expected := []struct {
			method  string
			url     string
			section string
			want    interface{}
		}{
			{"GET", "/", "code", 200},
			{"GET", "/", "body", "API is up"},
			{"GET", "/generate/a", "code", 200},
			{"GET", "/generate/b", "code", 200},
			{"GET", "/generate/b", "code", 200},
			{"GET", "/view/0000b", "code", 422},
			{"GET", "/g/20/a", "code", 404},
		}

		for i, test := range expected {
			t.Run(fmt.Sprintf("#%d: %q:%s", i, test.method, test.url), func(t *testing.T) {
				response := callHTTPEndpointHandler(t, test.method, test.url)
				switch test.section {
				case "code":
					checkError(t, response.Code, test.want, fmt.Sprintf("%q%q", test.method, test.url))
				case "body":
					assert.Contains(t, response.Body.String(), test.want.(string))

				}

			})
		}

	})

}

func TestStretchApiIdempotency(t *testing.T) {
	// Expected shortcode endings of the 16-digit hex to be produced
	// by the generator
	var a1, a2, a3 string = "0002F", "00032", "00064"

	t.Run("TEST idempotency", func(t *testing.T) {
		expected := []struct {
			method  string
			url     string
			section string
			want    interface{}
		}{
			{"GET", "/generate/a", "body", a1},
			{"GET", "/generate/a", "body", a2},
			{"GET", "/generate/b", "body", a3},
		}
		for i, test := range expected {

			t.Run(fmt.Sprintf("#%d: %q:%s", i, test.method, test.url), func(t *testing.T) {

				response := callHTTPEndpointHandler(t, test.method, test.url)
				switch test.section {
				case "code":
					checkError(t, response.Code, test.want, fmt.Sprintf("%q%q", test.method, test.url))
				case "body":
					assert.Contains(t, response.Body.String(), test.want.(string))
				}

			})
		}

	})

}

// DRY Helper method to check errors
func checkError(t *testing.T, got, want interface{}, reqPath string) {
	if got != want {
		t.Errorf("request %v: got %v, want %v", reqPath, got, want)
	}
}

// DRY Helper method to perform a http request on an endpoint
func callHTTPEndpointHandler(t *testing.T, httpMethod, url string) *httptest.ResponseRecorder {
	request, err := http.NewRequest(httpMethod, url, nil)
	assert.NilError(t, err)
	// create a http response (test recorder)
	response := httptest.NewRecorder() // in order to capture the http response

	// make a http request to the endpoint defined in our test suite
	// and capture the response in the recorder
	rt.ServeHTTP(response, request)
	return response
}
