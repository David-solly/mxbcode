package main

import (
	"log"
	"net/http"

	mockendpoint "github.com/David-solly/mxbcode/mock_lorawan_endpoint"
)

func main() {
	var errchan = make(chan error)
	go func() {
		errchan <- http.ListenAndServe(":8080", mockendpoint.GetLorawanRouter(false))
	}()
	log.Println(<-errchan)
}
