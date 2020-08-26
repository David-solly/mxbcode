package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"goprojects/MMAX/barcode-system/pkg/api/v1/cache"
	"goprojects/MMAX/barcode-system/pkg/api/v1/generator"
	"goprojects/MMAX/barcode-system/pkg/api/v1/models"

	"github.com/docker/docker/pkg/testutil/assert"
)

const tofMaxRequests = 10

// resets the listening server database
// for testing responses
//
func reset() {
	resp, err := http.Get(urlDebug)
	if err != nil {
		fmt.Print(err)
	}
	defer resp.Body.Close()

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
			reset()
			k := runGenerator(int64(test.want))
			assert.Contains(t, k, test.resp)

		})
	}
}

func TestInitCacheInMain(t *testing.T) {
	isPackageTest = true
	t.Run("Initialise cache", func(t *testing.T) {
		suite := []struct {
			testName string
			want     bool
			url      string
			err      string
		}{
			{"INITIALISE - ", true, "", ""},
			{"INITIALISE - ", true, "192.168.99.100:6379", ""},
			{"INITIALISE - ", false, "192.168.99.100:679", "refused"},
			{"INITIALISE - ", true, "", ""},
		}

		for i, test := range suite {
			reset()
			t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
				k, err := initCache(test.url)
				assert.DeepEqual(t, k, test.want)
				if test.err != "" {
					assert.Error(t, err, test.err)
				}
			})
		}
	})
}
func TestGenerateCli(t *testing.T) {
	// Using cache defined in main
	c := cache.Cache{}
	c.Initialise("", false)

	ch := make(chan bool)

	suite := []struct {
		testName  string
		want      int64
		shortcode string
		output    string
		err       string
	}{
		{"GENERATE - ", 1, "00001", "{\"deveuis\":[", ""},
		{"GENERATE - ", 100, "00002", "{\"deveuis\":[", ""},
		{"GENERATE - ", 80, "00070", "{\"deveuis\":[", ""},
	}

	for i, test := range suite {
		reset()
		t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
			uids, err := generateBatchIDs(test.want, c, ch)
			if test.err != "" {
				assert.Error(t, err, test.err)

			}

			if test.err == "" {
				assert.NilError(t, err)
				assert.Contains(t, uids, test.shortcode)
				assert.Contains(t, uids, test.output)
			}

		})
	}
}

func TestRegisterFromCli(t *testing.T) {

	//generate 100 unique ID's
	hundred, _ := generator.GenerateDUIDBatch(100, c.Client)
	hundred2, _ := generator.GenerateDUIDBatch(100, c.Client)
	hundred3, _ := generator.GenerateDUIDBatch(100, c.Client)
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
		{"Register - ", *hundred2, 100, "0006F\"", "OK", ""},
		{"Register - ", *hundred3, 100, "000CA\"", "OK", ""},
	}

	for i, test := range suite {
		reset()
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
			uids, count, err := registerBatch(test.data, c, ch)
			if test.err != "" {
				assert.Error(t, err, test.err)

			}

			if test.err == "" {
				assert.NilError(t, err)
				assert.Contains(t, uids, test.want)
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
	// reset()
	t.Run("Test main flow", func(t *testing.T) {
		suite := []struct {
			testName string
			cmd      string
			args     []string
			contains string
			err      string
		}{
			{"RUN CMD - ", "go", []string{"run", ".", "-count=10"}, "deveui", ""},
			{"RUN CMD - ", "go", []string{"run", "."}, "deveui", ""},
			{"RUN CMD - ", "g", []string{"run", ".", "-count=10"}, "deveui", "not found"},
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
					fmt.Println(str)
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
//
//
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

	for i, test := range suite {

		t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
			for reqs := 0; reqs < 100; reqs++ {
				tof <- i
				go func() {
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
}
