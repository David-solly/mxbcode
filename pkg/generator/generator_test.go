package generator

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/David-solly/mxbcode/pkg/cache"
	"github.com/David-solly/mxbcode/pkg/models"

	"github.com/docker/docker/pkg/testutil/assert"
)

var c = cache.Cache{}

func TestMain(t *testing.M) {
	fmt.Printf("Starting setup\n")

	c.Initialise("", false) // Initialise in-memory cache
	tmpData, _, _ := c.Client.ReadCache(cache.LastUIDKey)
	c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: "00000"})

	v := t.Run()

	fmt.Printf("\nFinishing teardown\n")
	c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: tmpData})

	if key, k := c.Client.(*cache.MemoryCache); k {
		key.Persist()

		c.Client.Initialise()
	}
	os.Exit(v)

}

//reset cache for testing purposes
func resetCache() {
	c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: "00000"})
}

// Generate 100 values
func TestGenerateBatchValues(t *testing.T) {
	dataType := &[]*models.DevEUI{}

	t.Run("GENERATE BATCH DevEUIs", func(t *testing.T) {
		suite := []struct {
			testName      string
			want          int
			lastShortcode string
			err           string
		}{
			{"GENERATE - 1", 1, "00001", ""},
			{"GENERATE - 4", 4, "00005", ""},
			{"GENERATE - 10", 10, "0000F", ""},
			{"GENERATE - 99", 99, "00072", ""},
			{"GENERATE - 100", 100, "000D6", ""},
			{"GENERATE - 123", 123, "000D6", "Too many requested - Maximum 100"},
			{"GENERATE - 0", 0, "000D6", "Minimum request is 1"},
			{"GENERATE - -1", -1, "000D6", "Minimum request is 1"},
			{"GENERATE - 2", 2, "<SOMETHING INVALID>", "invalid hexcode"},
			{"GENERATE - 88", 88, "FFFFE", "insufficient ID space (1) remaining to generate (88) IDs"},
		}
		for i, test := range suite {

			if i == len(suite)-1 { // Simulate sub-overflow condition
				c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: "ffffe"})
			}
			if i == len(suite)-2 { // Simulate data error
				c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: "<something invalid>"})
			}

			t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
				ids, err := GenerateDUIDBatch(test.want, c.Client)
				last, _, _ := c.Client.ReadCache(cache.LastUIDKey)
				assert.Equal(t, last, test.lastShortcode)

				if test.err == "" {
					dm := *ids
					assert.NilError(t, err)
					assert.NotNil(t, ids)
					assert.Equal(t, reflect.TypeOf(ids), reflect.TypeOf(dataType))
					assert.Equal(t, len(*ids), test.want)
					assert.Equal(t, strings.ToUpper(dm[test.want-1].ShortCode), test.lastShortcode)
					assert.Equal(t, len(dm[test.want-1].DevEUI) > 15, true)
				} else {
					assert.Error(t, err, test.err)
					assert.Equal(t, ids == nil, true)
				}
			})
		}

	})
}

func BenchmarkGenerateIDS(b *testing.B) {
	cache := cache.Cache{}
	cache.Initialise("", false)
	b.Run("GENERATE BATCH", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = GenerateDUIDBatch(100, cache.Client)
		}
	})
}
