package main

import (
	"log"
	"net/http"

	mockendpoint "github.com/David-solly/mxbcode/cmd/client/mock_endpoint"
)

func main() {
	var errchan = make(chan error)
	go func() {
		errchan <- http.ListenAndServe(":8080", mockendpoint.GetRouter(false))
	}()
	log.Println(<-errchan)
}
