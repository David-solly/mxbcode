package mockendpoint

import (
	"fmt"
	"time"

	"github.com/David-solly/mxbcode/pkg/cache"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var DB = &cache.Cache{}

// GetLorawanRouter :
// Returns the mock lorawan endpoint http server router
func GetLorawanRouter(silent bool) *chi.Mux {
	DB.Initialise("", false)
	fmt.Println("\ngetting mock router")
	r := chi.NewRouter()
	if !silent {
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		r.Use(middleware.Throttle(11))
	}

	r.Use(middleware.Timeout(1 * time.Second))
	r.Get("/", clearLorawanDatabase)
	r.Post("/", baseOK)
	r.Post("/sensor-onboarding-sample", registerEndpoint)

	return r

}
