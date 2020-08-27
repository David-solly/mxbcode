package main

import (
	mockendpoint "goprojects/MMAX/barcode-system/cmd/client/mock_endpoint"
	"log"
	"net/http"
)

func main() {
	var errchan = make(chan error)
	go func() {
		errchan <- http.ListenAndServe(":8080", mockendpoint.GetRouter(false))
	}()
	log.Println(<-errchan)
}
