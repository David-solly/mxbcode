package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-chi/chi"
)

// Transforms the map to json byte slice
func toJSON(key string, value string) []byte {
	data := map[string]interface{}{key: value}
	if data == nil {
		return nil
	}

	dt, _ := json.Marshal(data)
	return dt
}

// DRY method to write to output
func write(w http.ResponseWriter, data []byte, code int) {
	w.WriteHeader(code)
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(data))
}

// Used to create a lookup key to check for cached results
// Useful for idempotency
func createRequestIDKey(r *http.Request) (uniqeResponseKey string) {
	reqID := chi.URLParam(r, "uid")
	urid := sha1.New()
	urid.Write([]byte(fmt.Sprintf("%s%s%s", r.UserAgent(), r.URL.Hostname(), reqID)))
	return fmt.Sprintf("%x", urid)

}

// Validates that a supplied shortcode
// meets the criteria before being processed
func shortcodeValidator(w http.ResponseWriter, sc string) bool {
	validHex, err := regexp.MatchString(`^[a-fA-F0-9]{1,5}$`, sc) //validate 5 digit hex
	if err != nil || !validHex {
		errorMessage := fmt.Sprintf("invalid shortcode - %v", sc)
		write(w, toJSON("error", errorMessage), http.StatusUnprocessableEntity)
		return false
	}

	return true
}
