package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/David-solly/mxbcode/pkg/models"
)

// LastUIDKey :
// Key for lookup of last generated UID in the cache store
var LastUIDKey = strings.ToUpper("last-deveui")

var persistFile = "persistfilefff-11-ff.dat"

type MemoryCache struct {
	client *Store
}

type Store struct {
	name  string
	data  map[string]string
	mutex sync.Mutex
}

func (c MemoryCache) NewClient() *Store {
	return &Store{name: "Memory store",
		data: map[string]string{"PING": "PONG", LastUIDKey: "00000"}}
}

func (c *MemoryCache) init() (string, error) {
	c.client = c.NewClient()

	data, err := ioutil.ReadFile(persistFile)
	if err == nil {
		dta := make(map[string]string)
		if err := json.Unmarshal(data, &dta); err != nil {
			fmt.Print("Reset count - ")
		} else {
			fmt.Printf("Restarted ")
			c.client.data[LastUIDKey] = dta[LastUIDKey]
		}

	}

	pong, _ := c.client.data["PING"]

	fmt.Println("Memory store - Online ..........")
	return pong, nil
}

func (c *MemoryCache) Initialise() (string, error) {
	return c.init()
}

func (c *MemoryCache) StoreDUID(model models.DevEUI) (bool, error) {
	c.client.mutex.Lock()
	c.client.data[strings.ToUpper(model.ShortCode)] = strings.ToUpper(model.DevEUI)
	c.client.mutex.Unlock()
	return true, nil
}

func (c *MemoryCache) ReadCache(key string) (string, bool, error) {
	c.client.mutex.Lock()
	data, k := c.client.data[strings.ToUpper(key)]
	c.client.mutex.Unlock()
	if !k {
		return "", false, errors.New(fmt.Sprintf("Device id with shortcode: '%q' - Not Found", key))
	}
	return data, true, nil
}

func (c *MemoryCache) StoreLastDUID(model models.LastDevEUI) (bool, error) {
	c.client.mutex.Lock()
	c.client.data[LastUIDKey] = strings.ToUpper(model.ShortCode)
	c.client.mutex.Unlock()
	return true, nil
}

// StoreDUIDGenResponse :
//For caching generate results from the same client - idempotent cache store
//
// Creates a sleeping gorouting that will awake and delete
// stored cached response found with 'k' only after 'duration'
func (c *MemoryCache) StoreDUIDGenResponse(model models.ApiResponseCacheObject) (bool, error) {
	c.client.mutex.Lock()
	c.client.data[strings.ToUpper(model.Key)] = model.Response
	c.client.mutex.Unlock()
	go func(k string, duration time.Duration, c *MemoryCache) {
		time.Sleep(duration)
		c.client.mutex.Lock()
		delete(c.client.data, k)
		c.client.mutex.Unlock()
		return
	}(model.Key, model.Timeout, c)
	return true, nil
}

func (c *MemoryCache) Persist() error {
	data, _ := json.Marshal(map[string]string{LastUIDKey: c.client.data[LastUIDKey]})
	return ioutil.WriteFile(persistFile, data, 0777)
}
