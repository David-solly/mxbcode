package main

import (
	"log"
	"net/http"
)

var db = make(map[string]bool)
var errchan = make(chan error)

func main() {
	go func() {
		errchan <- http.ListenAndServe(":8080", getRouter())
	}()
	log.Println(<-errchan)
}
