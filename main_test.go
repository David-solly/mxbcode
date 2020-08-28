package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	mockendpoint "github.com/David-solly/mxbcode/mock_lorawan_endpoint"
	"github.com/David-solly/mxbcode/pkg/generator"
	"github.com/David-solly/mxbcode/pkg/models"

	"github.com/docker/docker/pkg/testutil/assert"
)

const tofMaxRequests = 10

// Test flag to manually enable testing of a
// redis instance
var testRedis = false

// resets the listening server database
// for testing responses
func reset() {
	resp, err := cl.Get(urlDebug)
	if err != nil {
		fmt.Print(err)
	}
	defer resp.Body.Close()

}

func TestMain(t *testing.M) {
	fmt.Printf("Starting setup\n")

	//Start mock registration endpoint server
	ts := httptest.NewServer(mockendpoint.GetLorawanRouter(true))
	cl = ts.Client() //global http client
	mockendpoint.DB.Initialise("", false)
	c.Initialise("", false) // Initialise in-memory cache

	tmp := url
	url = ts.URL + "/sensor-onboarding-sample"
	urlDebug = ts.URL

	//stretch api
	RequestCache.Initialise("", false)
	rt = GetRouter()

	v := t.Run()
	ts.Close()

	url = tmp
	fmt.Printf("\nFinishing teardown\n")
	os.Exit(v)

}

func TestRunCli(t *testing.T) {
	suite := []struct {
		testName string
		want     int
		resp     string
	}{
		{"GENERATE FULL - ", 60, "{\"deveuis\":[\""},
		{"GENERATE FULL - ", 123, ""},
	}

	for i, test := range suite {
		t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
			k := runGenerator(int64(test.want))
			assert.Contains(t, k, test.resp)

		})
	}
}

func TestInitCacheInMain(t *testing.T) {

	t.Run("Initialise cache", func(t *testing.T) {
		suite := []struct {
			testName string
			want     bool
			url      string
			err      string
		}{

			{"INITIALISE redis - ", true, "192.168.99.100:6379", ""},
			{"INITIALISE redis - ", false, "192.168.99.100:679", "refused"},
			{"INITIALISE - ", true, "", ""},
		}

		for i, test := range suite {
			// Skip checking of redis databsase during full package test
			if !testRedis && test.testName != "INITIALISE redis - " {
				t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
					k, err := initCache(test.url)
					assert.DeepEqual(t, k, test.want)
					if test.err != "" {
						assert.Error(t, err, test.err)
					}
				})
			}
		}
	})
}
func TestGenerateCli(t *testing.T) {
	reset()

	hk := url
	ch := make(chan bool)
	chi := make(chan int, 1)

	suite := []struct {
		testName  string
		want      int
		shortcode string
		output    string
		err       string
	}{
		{"GENERATE - 1", 1, "00001", "{\"deveuis\":[", ""},
		{"GENERATE - 10", 10, "0000D", "{\"deveuis\":[", ""},
		{"GENERATE - 11", 11, "00018", "{\"deveuis\":[", ""},
		{"GENERATE - 12", 12, "00024", "{\"deveuis\":[", ""},
		{"GENERATE - 12", 12, "00030", "{\"deveuis\":[", ""},
		{"GENERATE - 100", 100, "0006A", "{\"deveuis\":[", ""},
		{"GENERATE - -1", 0, "{}", "{}", ""},
	}

	for i, test := range suite {
		c.Client.Initialise()
		if test.testName == "GENERATE - 10" {
			// Pre register devices in generate range
			// should automatically generate new values to compensate
			// should return requested quantity
			chi <- i
			register("00005", hk, chi)
			chi <- i
			register("00022", hk, chi)
			chi <- i
			register("00007", hk, chi)
		}
		t.Run(fmt.Sprintf("\n#%d: %q", i, test.testName), func(t *testing.T) {
			generated, uids, err := generateBatchIDs(int64(test.want), c, ch)
			if test.err != "" {
				assert.Error(t, err, test.err)

			}

			if test.err == "" {
				assert.NilError(t, err)
				assert.Contains(t, uids, test.shortcode)
				assert.Contains(t, uids, test.output)
				assert.Equal(t, generated, test.want)
				if test.testName == "GENERATE - -1" {
					assert.Equal(t, len(uids), 2)
				} else {
					assert.Equal(t, len(uids), (18)*test.want+13+test.want)
				}

			}

		})
	}
}

func TestRegisterFromCli(t *testing.T) {

	//generate 100 unique ID's
	hundred, _ := generator.GenerateDUIDBatch(100, c.Client)
	ch := make(chan bool)
	suite := []struct {
		testName string
		data     []*models.DevEUI
		count    int
		want     string
		resp     string
		err      string
	}{
		{"Register - ", []*models.DevEUI{{DevEUI: "d19ef65832100001", ShortCode: "00001"}}, 1, "{\"deveuis\":[\"D19EF65832100001\"]}", "OK", ""},
		{"Register - ", []*models.DevEUI{
			{DevEUI: "d19ef65832100002", ShortCode: "00002"},
			{DevEUI: "d19ef65832100003", ShortCode: "00003"},
			{DevEUI: "d19ef65832100004", ShortCode: "00004"},
			{DevEUI: "d19ef65832100005", ShortCode: "00005"},
		}, 4, "\"D19EF65832100005\"", "OK", ""},
		{"Register - ", *hundred, 100, "0003D\"", "OK", ""},
	}

	for i, test := range suite {
		reset()
		c.Initialise("", false)
		if i == len(suite)-1 {
			go func() {
				// Simulate interrupt signal
				//
				time.Sleep(time.Duration(time.Millisecond * 15))
				interrupt <- true

			}()
		}
		if i == len(suite)-2 {
			go func() {
				// Simulate interrupt signal
				//
				time.Sleep(time.Duration(time.Millisecond * 1))
				ch <- true
			}()
		}

		t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
			uidlist, count, err := registerBatch(test.data, c, ch, &models.RegisteredDevEUIList{})
			uids, _ := json.Marshal(uidlist)
			if test.err != "" {
				assert.Error(t, err, test.err)

			}

			if test.err == "" {
				assert.NilError(t, err)
				assert.Contains(t, string(uids), test.want)
				if i == len(suite)-2 {
					assert.Equal(t, count != test.count, true)
				} else {
					assert.Equal(t, count, test.count)
				}
			}

		})
	}
}

func TestRegisterWithProvider(t *testing.T) {

	tof := make(chan int, tofMaxRequests)
	suite := []struct {
		testName  string
		shortcode string
		status    string
		code      int
		err       string
	}{
		{"REGISTER WITH PROVIDER - ", "FFFF1", "200 OK", 200, ""},
		{"REGISTER WITH PROVIDER - ", "FFFF2", "200 OK", 200, ""},
		// {"REGISTER WITH PROVIDER - ", "00001", "422 Unprocessable Entity", 422, ""},
	}

	for i, test := range suite {
		tof <- i
		reset()
		t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
			sc, result, err := register(test.shortcode, url, tof)
			if test.err != "" {
				assert.Error(t, err, test.err)
			} else {
				assert.DeepEqual(t, result, test.code)
				assert.DeepEqual(t, sc, test.status)
			}

		})
	}
}

func TestGenerateFromCMD(t *testing.T) {
	t.Run("Test main flow", func(t *testing.T) {
		suite := []struct {
			testName string
			cmd      string
			args     []string
			contains string
			err      string
		}{
			{"RUN CMD - ", "go", []string{"run", ".", "-count=10", "-reg-url=" + url}, "deveui", ""},
			{"RUN CMD - ", "go", []string{"run", ".", "-reg-url=" + url}, "deveui", ""},
			{"RUN CMD - ", "g", []string{"run", ".", "-count=10", "-reg-url=" + url}, "deveui", "not found"},
		}

		for i, test := range suite {
			reset()
			t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
				var out bytes.Buffer //to capture stdout output
				cmd := exec.Command(test.cmd, test.args...)
				cmd.Stdout = &out
				err := cmd.Run()
				if test.err != "" {
					assert.Error(t, err, test.err)
				} else {
					str := out.String()
					assert.NilError(t, err)
					assert.Contains(t, str, test.contains)

				}

			})
		}

	})
}

// Time of flight requests
// monitor current in flight requests
// decrement pool when request finishes
// fill pool when space available
// pass information through channels
func TestToFRequests(t *testing.T) {
	tofMaxRequests := 10
	tof := make(chan int, tofMaxRequests)
	suite := []struct {
		testName  string
		shortcode string
		status    string
		code      int
		err       string
	}{
		{"REGISTER WITH PROVIDER - ", "00001", "200 OK", 200, ""},
	}

	wg := sync.WaitGroup{}

	for i, test := range suite {

		t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
			for reqs := 0; reqs < 100; reqs++ {
				tof <- i
				go func() {
					defer wg.Done()
					wg.Add(1)
					sc, result, err := register(test.shortcode, urlDebug, tof)
					if test.err != "" {
						assert.Error(t, err, test.err)
					}
					if test.err == "" {
						assert.DeepEqual(t, result, test.code)
						assert.DeepEqual(t, sc, test.status)
					}
				}()

			}

		})
	}
	wg.Wait()
}

func TestMMaxFunction(t *testing.T) {
	t.Run("Test command line flags", func(t *testing.T) {
		suite := []struct {
			testName string
			contain  string
			flag     string
			value    string
			err      string
		}{
			// {"FLAG - -addr=localhost", "", "reg-url", "localhost", ""},
			{"FLAG - -l=12345", "", "l", "-4dcm", ""},
			{"FLAG - -l=12345", "123A1", "l", "12345", ""},
			{"FLAG - -count=10", "123AA", "count", "10", ""},
			{"FLAG - -count=0", "{}", "count", "0", ""},
			{"FLAG - -count=abcd", "", "count", "abcd", ""},
			// {"FLAG - -port=8085", "", "port", "8085", ""},
		}

		for i, test := range suite {
			t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
				flag.Set(test.flag, test.value)
				dta := mmax()
				assert.Contains(t, dta, test.contain)
			})
		}

	})

}
