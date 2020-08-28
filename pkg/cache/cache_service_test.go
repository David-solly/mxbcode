package cache

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/David-solly/mxbcode/pkg/models"

	"github.com/docker/docker/pkg/testutil/assert"
)

// set to a reachable redis instance if one is available
// otherwise ignored
const globalRedis = "192.168.99.100:6379"
const useRedis = false

func TestInitialiseCache(t *testing.T) {
	c := Cache{}
	t.Run("INITIALISE cache", func(t *testing.T) {
		suite := []struct {
			testName  string
			addr      string
			use, want bool
			cacheType Service
			err       string
		}{
			{"INITIALISE - redis", globalRedis, true, true, &RedisCache{}, ""},
			{"INITIALISE - redis", "192.168.99.100:6349", true, false, &RedisCache{}, "No connection"},
			{"INITIALISE - redis", globalRedis, false, true, &MemoryCache{}, ""},
			{"INITIALISE - none", "", true, false, nil, "No address supllied"},
			{"INITIALISE - memory", "", false, true, &MemoryCache{}, ""},
		}
		for i, test := range suite {
			if !useRedis && test.testName == "INITIALISE - redis" {
				fmt.Println("Skipping redis check")
				continue
			}
			t.Run(fmt.Sprintf("#%d: %q%q", i, test.testName, test.addr), func(t *testing.T) {
				ok, err := c.Initialise(test.addr, test.use)
				if test.err == "" {
					assert.NotNil(t, c.Client)
					assert.NilError(t, err)
					assert.Equal(t, ok, test.want)
					if !test.use {
						assert.Equal(t, reflect.TypeOf(c.Client), reflect.TypeOf(test.cacheType))
					}
					if test.use {
						assert.Equal(t, reflect.TypeOf(c.Client), reflect.TypeOf(test.cacheType))
					}

				} else {
					assert.Error(t, err, test.err)
					assert.Equal(t, ok, test.want)
				}
			})
		}
		t.Run("INITIALISE cache FAULT - redis", func(t *testing.T) {
			os.Setenv("REDIS_DSN", "")
			r := RedisCache{}
			s, e := r.Initialise()
			assert.Equal(t, s, "")
			assert.Error(t, e, "No address supplied")
		})

	})
}

func TestStoreGenerateResposne(t *testing.T) {
	c := Cache{}
	c.Initialise("", false)
	longDuration := time.Duration(time.Second * 10)
	shortDuration := time.Duration(time.Millisecond * 300)
	suite := []struct {
		testName string
		data     models.ApiResponseCacheObject
		expect   bool
		err      string
	}{
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722AA7", Response: "38240", Timeout: longDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A11", Response: "38245", Timeout: shortDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A11", Response: "38245", Timeout: shortDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A12", Response: "38245", Timeout: shortDuration}, true, ""},
	}

	for i, test := range suite {
		// Retrieve full value via 5 char shortcode
		t.Run(fmt.Sprintf("#%d - SAVE CACHE: %q", i, test.data.Key), func(t *testing.T) {
			k, err := c.Client.StoreDUIDGenResponse(test.data)
			assert.NilError(t, err)
			assert.DeepEqual(t, k, true)
			// device, found, err := c.Client.StoreDUIDGenResponse(test.data.ShortCode)
		})
	}

	suiteRead := []struct {
		testName string
		data     models.ApiResponseCacheObject
		expect   bool
		err      string
	}{
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722AA7", Response: "38240", Timeout: longDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A11", Response: "38245", Timeout: shortDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A111", Response: "", Timeout: shortDuration}, false, "Not Found"},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A12", Response: "38245", Timeout: shortDuration}, false, "Not Found"},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A11", Response: "", Timeout: shortDuration}, false, "Not Found"},
	}
	for i, test := range suiteRead {

		time.Sleep(100 * time.Millisecond)

		t.Run(fmt.Sprintf("#%d - READ CACHE: %q", i, test.data.Key), func(t *testing.T) {
			s, k, err := c.Client.ReadCache(test.data.Key)
			assert.DeepEqual(t, k, test.expect)
			if test.err != "" {
				assert.Error(t, err, test.err)
			} else {
				assert.NilError(t, err)
			}

			if k {
				assert.DeepEqual(t, s, test.data.Response)
			}

		})
	}
}

func TestStoreGenerateResposneREDIS(t *testing.T) {
	c := Cache{}
	c.Initialise(globalRedis, useRedis)
	longDuration := time.Duration(time.Second * 10)
	shortDuration := time.Duration(time.Millisecond * 300)
	suite := []struct {
		testName string
		data     models.ApiResponseCacheObject
		expect   bool
		err      string
	}{
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722AA7", Response: "38240", Timeout: longDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A11", Response: "38245", Timeout: shortDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A11", Response: "38245", Timeout: shortDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A12", Response: "38245", Timeout: shortDuration}, true, ""},
	}

	for i, test := range suite {
		// Retrieve full value via 5 char shortcode
		t.Run(fmt.Sprintf("#%d - SAVE CACHE: %q", i, test.data.Key), func(t *testing.T) {
			k, err := c.Client.StoreDUIDGenResponse(test.data)
			assert.NilError(t, err)
			assert.DeepEqual(t, k, true)

		})
	}

	suiteRead := []struct {
		testName string
		data     models.ApiResponseCacheObject
		expect   bool
		err      string
	}{
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722AA7", Response: "38240", Timeout: longDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A11", Response: "38245", Timeout: shortDuration}, true, ""},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A111", Response: "", Timeout: shortDuration}, false, "Not Found"},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A12", Response: "38245", Timeout: shortDuration}, false, "Not Found"},
		{"CACHE -  ", models.ApiResponseCacheObject{Key: "FFA45722A11", Response: "", Timeout: shortDuration}, false, "Not Found"},
	}
	for i, test := range suiteRead {

		time.Sleep(100 * time.Millisecond)

		t.Run(fmt.Sprintf("#%d - READ CACHE: %q", i, test.data.Key), func(t *testing.T) {
			s, k, err := c.Client.ReadCache(test.data.Key)
			assert.DeepEqual(t, k, test.expect)
			if test.err != "" {
				assert.Error(t, err, test.err)
			} else {
				assert.NilError(t, err)
			}

			if k {
				assert.DeepEqual(t, s, test.data.Response)
			}

		})
	}
}

func TestCache5chars(t *testing.T) {
	mCache := Cache{}
	mCache.Initialise("", false)

	rCache := Cache{}
	// Redis endpoint - true flag to confirm redis as choice
	rCache.Initialise(globalRedis, useRedis)

	t.Run("SAVE and READ from CACHE", func(t *testing.T) {
		suite := []struct {
			testName string
			data     models.DevEUI
			expect   bool
			cache    Cache
			err      string
		}{
			{"CACHE - memory ", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38240"}, true, mCache, ""},
			{"CACHE - redis", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38241"}, true, rCache, ""},
			{"CACHE - redis", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38251"}, false, Cache{Client: &RedisCache{}}, ""},
			{"CACHE - memory", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38242"}, true, mCache, ""},
			{"CACHE - memory", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38212"}, true, mCache, ""},
		}

		for i, test := range suite {
			t.Run(fmt.Sprintf("#%d - %q: %q", i, test.testName, test.data.ShortCode), func(t *testing.T) {

				ok, err := test.cache.Client.StoreDUID(test.data)
				assert.Equal(t, ok, test.expect)
				if !test.expect {
					assert.Error(t, err, test.err)
				} else {
					assert.NilError(t, err)
				}
			})
		}

		t.Run("READ from CACHE", func(t *testing.T) {
			suite := []struct {
				testName string
				data     models.DevEUI
				expect   bool
				cache    Cache
				err      string
			}{
				{"CACHE - memory ", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38240"}, true, mCache, ""},
				{"CACHE - redis", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38241"}, true, rCache, ""},
				{"CACHE - memory", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38225"}, false, mCache, "Device id with shortcode: '\"38225\"' - Not Found"},
				{"CACHE - memory", models.DevEUI{DevEUI: "FFA45722AA7", ShortCode: "38225"}, false, rCache, "Device id with shortcode: '\"38225\"' - Not Found"},
			}

			for i, test := range suite {
				// Retrieve full value via 5 char shortcode
				t.Run(fmt.Sprintf("#%d - READ CACHE: %q", i, test.data.ShortCode), func(t *testing.T) {
					device, found, err := test.cache.Client.ReadCache(test.data.ShortCode)
					assert.Equal(t, found, test.expect)
					assert.Equal(t, reflect.TypeOf(device), reflect.TypeOf("deveui"))
					if test.expect {
						assert.NilError(t, err)
						assert.Equal(t, device, test.data.DevEUI)
					}
					if !test.expect {
						assert.Error(t, err, test.err)
						assert.Equal(t, device, "")

					}

				})
			}

		})
	})
}

func TestLastIDCache(t *testing.T) {
	mCache := Cache{}
	mCache.Initialise("", false)

	rCache := Cache{}
	rCache.Initialise(globalRedis, useRedis)
	t.Run("SAVE last id to cache", func(t *testing.T) {
		suite := []struct {
			testName string
			data     models.LastDevEUI
			expect   bool
			cache    Cache
			err      string
		}{
			{"CACHE - memory ", models.LastDevEUI{ShortCode: "38240"}, true, mCache, ""},
			{"CACHE - redis ", models.LastDevEUI{ShortCode: "38240"}, true, rCache, ""},
		}

		for i, test := range suite {
			t.Run(fmt.Sprintf("#%d - %q: %q", i, test.testName, test.data.ShortCode), func(t *testing.T) {
				ok, err := test.cache.Client.StoreLastDUID(test.data)
				assert.Equal(t, ok, test.expect)
				if !test.expect {
					assert.Error(t, err, test.err)
				} else {
					assert.NilError(t, err)
				}
			})
		}
	})
}
