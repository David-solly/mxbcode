package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goprojects/MMAX/barcode-system/pkg/api/v1/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/pkg/testutil/assert"
)

// Tripped when running package test
// maps values accordingly
//
var isPackageTest = false

func TestRouter(t *testing.T) {
	tmpURL := url
	isPackageTest = true
	t.Run("TEST routes and functions", func(t *testing.T) {
		// register dummy device
		// to be read back by endpoint
		//
		dui := models.DevEUI{DevEUI: "d19ef65832100001", ShortCode: "00001"}
		c.Client.StoreDUID(dui)
		// define endpoint methods
		// smaple body
		//
		// map[string]string{"shortcode": "00001"}
		//
		expected := []struct {
			method  string
			url     string
			body    map[string]string
			section string
			want    interface{}
		}{
			{"GET", "/", nil, "code", 200},
			{"GET", "/", nil, "body", "Api is up"},
			{"GET", "/view/00001", nil, "code", 200},
			{"GET", "/view/00001", nil, "body", "00001"},
			{"GET", "/view/00000", nil, "body", "Not Found"},
			{"GET", "/view/<something invalid>", nil, "body", "invalid shortcode"},
			{"GET", "/view/<something invalid>", nil, "code", 422},
			{"GET", "/activate/00001", nil, "code", 200},
			{"GET", "/activate/-54-", nil, "body", "invalid shortcode"},
			{"GET", "/activate/0ff01", nil, "code", 200},
			{"GET", "/activate/0000", nil, "body", "Successfully Registered"},
			{"GET", "/activate/0000", nil, "body", "Already Activated"},
			{"GET", "/activate/0000", nil, "code", 422},
			{"GET", "/generate/20/a", nil, "code", 200},
			{"GET", "/generate/120/a", nil, "code", 200},
			{"GET", "/generate/-1/a", nil, "code", 422},
			{"GET", "/generate/-1/a", nil, "body", "minimum is 1"},
			{"GET", "/generate/a120/a", nil, "code", 422},
			{"GET", "/g/20/a", nil, "code", 404},
			{"GET", "/activate/0ff01", nil, "code", 500}, //remove global url to simulate server crash error
		}

		rt := getRouter()

		for i, test := range expected {
			t.Run(fmt.Sprintf("#%d: %q", i, test.method), func(t *testing.T) {
				if i == len(expected)-1 {
					// out of ip range
					url = "192.168.1.256:123456"
				}
				body, _ := json.Marshal(test.body)
				// create a request
				request, err := http.NewRequest(test.method, test.url, bytes.NewBuffer(body))
				assert.NilError(t, err)
				// create a http response (test recorder)
				response := httptest.NewRecorder()
				// call endpoint
				rt.ServeHTTP(response, request)

				switch test.section {
				case "code":
					checkError(t, response.Code, test.want, fmt.Sprintf("%q%q", test.method, test.url))
				case "body":
					assert.Contains(t, response.Body.String(), test.want.(string))

				}

			})
		}

	})
	url = tmpURL //fix broken url

}

func TestIdempotencyRouter(t *testing.T) {
	// Requires server to be running before testing
	// url endpoint
	//
	//urlE := "http://127.0.0.1:8082"

	// Context switching values depending on test run
	// full package test tends to have higher values due
	// to live caching
	//
	var a1, a2, a3 string = "0002F", "00032", "00078"

	t.Run("TEST idempotency", func(t *testing.T) {
		expected := []struct {
			method  string
			url     string
			body    map[string]string
			section string
			want    interface{}
		}{
			{"GET", "/generate/50/a", nil, "body", a1},
			{"GET", "/generate/50/a", nil, "body", a2},
			{"GET", "/generate/70/a", nil, "body", a3},
			{"GET", "/generate/70/a", nil, "body", a3},
		}

		rt := getRouter()

		for i, test := range expected {
			t.Run(fmt.Sprintf("#%d: %q", i, test.method), func(t *testing.T) {
				body, _ := json.Marshal(test.body)
				// create a request
				request, err := http.NewRequest(test.method, test.url, bytes.NewBuffer(body))
				assert.NilError(t, err)
				// create a http response (test recorder)
				response := httptest.NewRecorder()
				// call endpoint
				rt.ServeHTTP(response, request)

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

func checkError(t *testing.T, got, want interface{}, reqPath string) {
	if got != want {
		t.Errorf("request %v: got %v, want %v", reqPath, got, want)
	}
}
