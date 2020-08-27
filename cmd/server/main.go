package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"goprojects/MMAX/barcode-system/pkg/api/v1/cache"
	gen "goprojects/MMAX/barcode-system/pkg/api/v1/generator"
	"goprojects/MMAX/barcode-system/pkg/api/v1/models"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var (
	// Application cache /store
	//
	c cache.Cache

	// default http client
	// used for making requests to server
	// global timeout available
	//
	cl = &http.Client{}

	// Channels to interupt the program and signal
	// when generation is complete
	//
	interrupt = make(chan interface{})
	complete  = make(chan bool)
	shutdown  = make(chan bool)
	// API registration endpoint
	//
	// url      string = "http://127.0.0.1:8080/sensor-onboarding-sample"
	urlDebug string = "http://127.0.0.1:8080/"
	url      string = "http://europe-west1-machinemax-dev-d524.cloudfunctions.net/sensor-onboarding-sample"
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

	// initialise and in memory store
	// can be substituted for any data store
	// satisfying the `Service` interface defined in
	// pkg/api/v1/cache/cache_service.go
	if addr == "" {
		return c.Initialise("", false)
	}

	//REDIS url to bind to connect to
	//
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

	// initialise the cahe accordingly
	// if address suplied - Redis
	// otherwise in memory
	//
	initCache(*redis)

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
	//
	if *port != "" && len(*port) >= 1 {

		go http.ListenAndServe(*addr+":"+*port, getRouter())
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
	rd := 0
	registered := models.RegisteredDevEUIList{
		DevEUIs: []string{},
	}

	defer func() {
		uids, _ := json.Marshal(registered)
		data = string(uids)
	}()

	for int64(len(registered.DevEUIs)) < count {
		select {
		case <-ch:
			count = -1
		default:
			ids, e := gen.GenerateDUIDBatch(int(count)-len(registered.DevEUIs), c.Client)
			if e != nil {
				return generated, data, e
			}

			registered, rd, err = registerBatch(*ids, c, ch, &registered)
			if err != nil {
				return
			}
			generated += rd
		}
	}

	return
}

// loop through generated id's
// register each one concurrently
// keep track of count of routines
// listen for SIGINT and return current tally
//
func registerBatch(batch []*models.DevEUI, c cache.Cache, sigint chan bool, registered *models.RegisteredDevEUIList) (models.RegisteredDevEUIList, int, error) {

	// monitor duplicates
	//
	watch := make(map[string]string)
	m := sync.Mutex{}

	tofMaxRequests := 10
	tof := make(chan int, tofMaxRequests)

	// Desired metrics
	// output should match this amount
	//

	var wg sync.WaitGroup

	for i, deveui := range batch {
		select {
		case _ = <-sigint:
			// Wait for inflight requests to finish
			// monitor buffer channel until depleted
			//
			wg.Wait()
			fmt.Println("Generated and registered ", len(registered.DevEUIs))

			// Output the completed and registered devices
			// to the user
			// save current registration point to cache
			//
			c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: deveui.ShortCode})
			return *registered, len(watch), nil

		default:
			// sleep is a debug feature
			// used to slow down requests to simulate
			// slow connection and fill up the
			// request buffer
			// used to test ToF and graceful interrupt
			//
			// time.Sleep(100 * time.Millisecond)

			// Start filling the buffered channel with values
			// blocks when there are 10 and waits for free space
			// maximum value can be adjusted accordingly
			// Maximum of 10 concurrent requests as per spec
			//
			tof <- i

			// concurrently send requests to register device ids
			//
			wg.Add(1)
			go func(deveui *models.DevEUI) {
				defer wg.Done()
				// in++
				// request parameters are in upper case hex as per request
				//
				_, code, err := register(strings.ToUpper(deveui.ShortCode), url, tof)
				if err != nil {
					fmt.Printf("Error registering %q:\n%s", deveui.ShortCode, err.Error())
				}

				if code == 200 {
					// c.Client.StoreDUID(*deveui)
					m.Lock()
					c.Client.StoreDUID(*deveui)
					watch[strings.ToUpper(deveui.ShortCode)] = strings.ToUpper(deveui.DevEUI)
					registered.DevEUIs = append(registered.DevEUIs, strings.ToUpper(deveui.DevEUI))
					m.Unlock()
				}

			}(deveui)
		}

	}

	// Wait for inflight requests to finish
	// monitor buffer channel until depleted
	//
	wg.Wait()

	fmt.Println("Generated and registered ", len(registered.DevEUIs))
	return *registered, len(watch), nil
}

// method that sends the request to the endpoint
// deals with registering the 5 character code
// pops the in flight queue when done `<-ch`
//
func register(shortcode, url string, ch chan int) (string, int, error) {
	// pop item from channel buffer on completion
	//
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
