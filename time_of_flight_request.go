package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/David-solly/mxbcode/pkg/cache"
	"github.com/David-solly/mxbcode/pkg/models"
)

// loop through generated id's
// register each one concurrently
// keep track of count of routines
// listen for SIGINT and return current tally
//
func registerBatch(batch []*models.DevEUI, c cache.Cache, sigint chan bool, registered *models.RegisteredDevEUIList) (models.RegisteredDevEUIList, int, error) {
	m := sync.Mutex{}

	tofMaxRequests := 10
	tof := make(chan int, tofMaxRequests)

	var wg sync.WaitGroup

	for i, deveui := range batch {
		select {
		case <-sigint:
			// Wait for inflight requests to finish monitor buffer channel until depleted
			wg.Wait()
			fmt.Println("Generated and registered ", len(registered.DevEUIs))

			// Output the completed and registered devices to the user
			// save current shortcode value to cache
			c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: deveui.ShortCode})
			return *registered, len(registered.DevEUIs), nil

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
			tof <- i

			wg.Add(1)

			// concurrently send requests to register device ids
			go func(deveui *models.DevEUI) {
				defer wg.Done()

				// request parameters are in upper case hex as per request
				_, code, err := register(strings.ToUpper(deveui.ShortCode), url, tof)
				if err != nil {
					fmt.Printf("Error registering %q:\n%s", deveui.ShortCode, err.Error())
				}

				if code == 200 {
					m.Lock()
					c.Client.StoreDUID(*deveui)
					registered.DevEUIs = append(registered.DevEUIs, strings.ToUpper(deveui.DevEUI))
					m.Unlock()
				}

			}(deveui)
		}

	}

	// Wait for all inflight requests to finish
	wg.Wait()

	return *registered, len(registered.DevEUIs), nil
}
