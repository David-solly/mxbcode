package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func getRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Throttle(11))

	// Set request timeout
	// long enough
	r.Use(middleware.Timeout(1 * time.Second))
	r.Get("/", base)
	r.Post("/", base)
	r.Post("/sensor-onboarding-sample", registerEndpoint)

	return r

}

func registerEndpoint(w http.ResponseWriter, r *http.Request) {
	reqs := make(map[string]string)
	p, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(p, &reqs)

	//check db
	_, k := db[reqs["deveui"]]
	if k {
		w.WriteHeader(422)
		w.Write([]byte("already registered"))

		return
	}
	w.Write([]byte("OK"))
	db[reqs["deveui"]] = true

}

func base(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
	db = make(map[string]bool)
}
