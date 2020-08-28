package mockendpoint

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/David-solly/mxbcode/pkg/models"
)

func registerEndpoint(w http.ResponseWriter, r *http.Request) {
	reqs := make(map[string]string)
	p, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(p, &reqs)

	//check DB
	_, k, _ := DB.Client.ReadCache(reqs["deveui"])
	if k {
		w.WriteHeader(422)
		w.Write([]byte("already registered"))

		return
	}
	w.Write([]byte("OK"))
	DB.Client.StoreDUIDGenResponse(models.ApiResponseCacheObject{
		Key:      reqs["deveui"],
		Response: "true", Timeout: time.Duration(time.Hour * 1)})

}

func clearLorawanDatabase(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
	DB.Initialise("", false)
}

func baseOK(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
