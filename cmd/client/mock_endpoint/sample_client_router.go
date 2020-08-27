package mockendpoint

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/David-solly/mxbcode/pkg/api/v1/cache"
	"github.com/David-solly/mxbcode/pkg/api/v1/models"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var db = &cache.Cache{}

// GetRouter :
// Returns the endpoint of the router to mock a
// registration endpoint
//
func GetRouter(silent bool) *chi.Mux {
	db.Initialise("", false)
	fmt.Println("\ngetting mock router")
	r := chi.NewRouter()
	if !silent {
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		r.Use(middleware.Throttle(11))
	}

	// Set request timeout
	// long enough
	r.Use(middleware.Timeout(1 * time.Second))
	r.Get("/", base)
	r.Post("/", baseOK)
	r.Post("/sensor-onboarding-sample", registerEndpoint)

	return r

}

func registerEndpoint(w http.ResponseWriter, r *http.Request) {
	reqs := make(map[string]string)
	p, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(p, &reqs)

	//check db
	_, k, _ := db.Client.ReadCache(reqs["deveui"])
	if k {
		w.WriteHeader(422)
		w.Write([]byte("already registered"))

		return
	}
	w.Write([]byte("OK"))
	db.Client.StoreDUIDGenResponse(models.ApiResponseCacheObject{
		Key:      reqs["deveui"],
		Response: "true", Timeout: time.Duration(time.Hour * 1)})

}

func base(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
	db.Initialise("", false)
}
func baseOK(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
