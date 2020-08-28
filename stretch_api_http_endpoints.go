package main

import (
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// GetRouter : Returns the mux containing the stretch API endpoints
func GetRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(2000 * time.Millisecond))

	// Generates a list of 100 id's
	// the reqID can be any value to make differentiate requests
	// you can cycle up from 1 to infinity if you like
	r.Get("/generate/{reqID}", GenerateBatchHTTPHandler)

	// retrieve a full 16 digit HEX device id  from the 5 digit 'shortcode'
	// if one does not exist. An appropriate message is returned
	r.Get("/view/{shortcode}", LookupShortcodeHTTPHandler)

	//check the basic status of the API
	r.Get("/", StatusHTTPHandler)

	return r

}
