package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"

	"syscall"

	"github.com/David-solly/mxbcode/pkg/cache"
	gen "github.com/David-solly/mxbcode/pkg/generator"
	"github.com/David-solly/mxbcode/pkg/models"
)

var (
	// Application cache /store
	c cache.Cache

	// default http client
	// used for making requests to server
	// global timeout available
	cl = &http.Client{}

	// Channels to interupt the program and signal
	// when generation is complete
	//
	interrupt = make(chan interface{})
	complete  = make(chan bool, 2)
	shutdown  = make(chan bool)

	shouldExit bool = false

	// LoRaWAN server endpoint or mock endpoint location
	url      string = "http://127.0.0.1:8080/sensor-onboarding-sample"
	urlDebug string = "http://127.0.0.1:8080/"
	// url      string = "http://europe-west1-machinemax-dev-d524.cloudfunctions.net/sensor-onboarding-sample"
)

// parse variable from commandline or exec command
//
var (
	last  = flag.String("l", "", "Explicitly set the last shortcode of previous batch.\nThe next batch will begin from here")
	reg   = flag.String("reg-url", "", "The registration endpoint url- \ndefault used if none is supplied")
	count = flag.String("count", "", "Number of DevEUIs to generate")
	addr  = flag.String("addr", "", "Bind address")
	port  = flag.String("port", "", "Bind port")
	redis = flag.String("redis-addr", "", "The address of the redis instance to use as a datacahe store")
)

// Init a cache
// Initial REDIS url to bind to connect to
// if blank, defaults to in memory cache
//
func init() {
	// set client to in memory first
	// aids in testing
	//
	c.Initialise("", false)
}
func initCache(addr string) (bool, error) {

	// initialise an in memory store
	// can be substituted for any data store
	// satisfying the `Service` interface defined in
	// pkg/cache/cache_service.go
	if addr == "" {
		return c.Initialise("", false)
	}

	//REDIS url to bind to if supplied
	return c.Initialise(addr, true)
}

func main() {
	mmax()
}

func mmax() (dta string) {
	// default value to generate set to 100
	//
	var idCount int64 = gen.DefaultMaxToGenerate

	flag.Parse() // parse supplied flags

	if *count != "" {
		def, err := strconv.ParseInt(*count, 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		idCount = def

	}

	// initialise the cahe accordingly-if address suplied - Redis
	// otherwise in-memory
	initCache(*redis)
	RequestCache = c

	if *last != "" {
		validHex, _ := regexp.MatchString(`^[a-fA-F0-9]{1,5}$`, *last)
		if !validHex {
			fmt.Printf("Invalid starting shortcode %q provided - exiting!", *last)
			return
		}
		c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: *last})

	}

	if *reg != "" {
		url = *reg
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		interrupt <- fmt.Errorf("%s", <-c)
	}()

	// run the http endpoint if the supplied flags match
	if *port != "" && len(*port) >= 1 {

		go http.ListenAndServe(*addr+":"+*port, GetRouter())
		<-interrupt
		log.Print("shutting down - server")
		return
	}

	return runGenerator(idCount)
}

func runGenerator(idCount int64) string {

	fmt.Println("MMAX - BATCH DevEUI Generator")
	var data string = ""
	// Maximum default value to generate if no commandline args
	// are provided
	//
	go func() {
		_, s, err := generateBatchIDs(idCount, c, shutdown)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("\n%s\n", s)
		data = s
		complete <- true
	}()

	select {
	case <-interrupt:
		{
			fmt.Println("\nGraceful shutdown...")
			shouldExit = true
			shutdown <- true

			<-complete

		}
	case <-complete:
		{
			return data
		}

	}
	return data
}

func generateBatchIDs(count int64, c cache.Cache, ch chan bool) (generated int, data string, err error) {
	registered := models.RegisteredDevEUIList{
		DevEUIs: []string{},
	}

	defer func() {
		generated = len(registered.DevEUIs)
		uids, _ := json.Marshal(registered)
		data = string(uids)
		fmt.Println("Generated and registered ", len(registered.DevEUIs))
		if generated > 0 {
			if c, k := c.Client.(*cache.MemoryCache); k {
				c.Persist()
			}

		}
	}()

	for int64(len(registered.DevEUIs)) < count && !shouldExit {

		ids, e := gen.GenerateDUIDBatch(int(count)-len(registered.DevEUIs), c.Client)
		if e != nil {
			return generated, data, e
		}

		if count == 0 {
			return
		}

		registered, _, err = registerBatch(*ids, c, ch, &registered)
		if err != nil {
			return
		}

	}

	return
}

// method that sends the request to the endpoint
// deals with registering the 5 character code with the LoRaWAN provider
// pops the in-flight queue when done `<-ch`
func register(shortcode, url string, ch chan int) (string, int, error) {
	// pop item from channel buffer on completion
	// used to let the time of flight counter to decrease
	defer func() {
		<-ch
	}()

	body, e := json.Marshal(map[string]string{"deveui": shortcode})
	if e != nil {
		return "Internal Server Error 500", 500, e
	}

	// create new request parameters
	// json as per requirement
	// posted in the body
	//
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {

		return "Internal Server Error 500", 500, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send and read the response body
	//
	resp, err := cl.Do(req)
	if e != nil {
		return "Bad Request 400", 400, err
	}

	if resp == nil {
		return "Internal Server Error 500", 500, errors.New("Blank response - check url is correct")
	}

	return resp.Status, resp.StatusCode, nil
}
