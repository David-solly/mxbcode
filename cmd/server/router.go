package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"goprojects/MMAX/barcode-system/pkg/api/v1/models"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

	// Set request timeout
	// long enough
	r.Use(middleware.Timeout(2000 * time.Millisecond))
	r.Get("/", base)
	r.Get("/view/{shortcode}", view)
	r.Get("/activate/{shortcode}", activate)
	r.Get("/generate/{count}/{uid}", generate)

	return r

}

func base(w http.ResponseWriter, r *http.Request) {
	write(w, toJSON(map[string]interface{}{
		"message": "Api is up",
	}))
}

// Idempotent generate function
// each request carries a unqe key
// the request params are hashed and the response is stored
// future requests with that key and number
// will return the same results.
//
func generate(w http.ResponseWriter, r *http.Request) {
	var val int64 = 100
	m := sync.Mutex{}
	reqID := chi.URLParam(r, "uid")
	count := chi.URLParam(r, "count")

	if len(count) > 3 {
		w.WriteHeader(422)
		write(w, toJSON(map[string]interface{}{
			"error": "invalid request value - maximum is 100",
		}))
		return
	}
	if count != "" {
		def, err := strconv.ParseInt(count, 10, 64)
		if err != nil || def < 1 {
			w.WriteHeader(422)
			write(w, toJSON(map[string]interface{}{
				"error": "invalid request value - minimum is 1 and maximum is 100",
			}))

			return

		}
		val = def

	}
	urid := sha1.New()
	urid.Write([]byte(fmt.Sprintf("%s%s%s%s", r.UserAgent(), r.URL.Hostname(), count, reqID)))
	requestKey := fmt.Sprintf("%x", urid)
	// read cached data first before generating new data
	//

	m.Lock()
	rq, k, _ := c.Client.ReadCache(requestKey)
	m.Unlock()
	// if err != nil {
	// 	w.WriteHeader(500)
	// 	w.Write([]byte(err.Error()))
	// 	return
	// }

	if k {
		write(w, []byte(rq))
		return
	}

	data := runGenerator(val)

	// Time limit to keep the response cached for
	// a few minutes
	//
	cacheDuration := time.Duration(time.Second * 720)
	m.Lock()
	c.Client.StoreDUIDGenResponse(models.ApiResponseCacheObject{Key: requestKey, Response: data, Timeout: cacheDuration})
	m.Unlock()

	write(w, []byte(data))
}

func activate(w http.ResponseWriter, r *http.Request) {
	sc := chi.URLParam(r, "shortcode")

	if k := shortcodeValidate(w, sc); !k {
		return
	}

	st, k, _ := c.Client.ReadCache(sc)
	if k {
		write(w, toJSON(map[string]interface{}{
			"deveui":  st,
			"message": "already active",
		}))
		return
	}
	ch := make(chan int, 1)
	ch <- 1
	st, _, e := register(sc, url, ch)

	if e != nil {

		w.WriteHeader(http.StatusInternalServerError)
		write(w, toJSON(map[string]interface{}{
			"error": fmt.Sprintf("error trying to register shortcode :%v ,", st),
		}))
		return
	}

	if strings.Contains(st, "Unprocessable Entity") {

		w.WriteHeader(422)
		write(w, toJSON(map[string]interface{}{
			"alert": fmt.Sprintf("Already Activated :%v ,", sc),
		}))
		return
	}
	if strings.Contains(st, "OK") {

		write(w, toJSON(map[string]interface{}{
			"message": fmt.Sprintf("Successfully Registered shortcode - %v", sc),
		}))
		return
	}
	w.Write([]byte(fmt.Sprintf("%v", st)))
}

func view(w http.ResponseWriter, r *http.Request) {
	sc := chi.URLParam(r, "shortcode")

	if k := shortcodeValidate(w, sc); !k {
		return
	}

	st, k, _ := c.Client.ReadCache(sc)
	if !k {
		write(w, toJSON(map[string]interface{}{
			"error": fmt.Sprintf("shortcode - %v is Not Found", sc),
		}))
		return
	}

	write(w, toJSON(map[string]interface{}{
		"deveui": st,
	}))
}

func toJSON(data map[string]interface{}) []byte {
	if data == nil {
		return nil
	}
	dt, _ := json.Marshal(data)
	return dt
}

func write(w http.ResponseWriter, data []byte) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(data))
}

func shortcodeValidate(w http.ResponseWriter, sc string) bool {
	validHex, err := regexp.MatchString(`^[a-fA-F0-9]{1,5}$`, sc)

	if err != nil || !validHex {

		w.WriteHeader(http.StatusUnprocessableEntity)
		write(w, toJSON(map[string]interface{}{
			"error": fmt.Sprintf("invalid shortcode - %v", sc),
		}))
		return false
	}

	return true
}
