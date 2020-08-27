package generator

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/David-solly/mxbcode/pkg/api/v1/cache"
	"github.com/David-solly/mxbcode/pkg/api/v1/models"

	"github.com/docker/docker/pkg/testutil/assert"
)

// Generate 100 values
func TestGenerateValues(t *testing.T) {
	c := cache.Cache{}
	c.Initialise("", false)

	suite := []struct {
		testName  string
		want      int
		shortcode string
		err       string
	}{
		{"GENERATE - ", 16, "00001", ""},
		{"GENERATE - ", 16, "00002", ""},
		{"GENERATE - ", 16, "00002", "invalid hexcode"},
	}

	for i, test := range suite {
		if i == len(suite)-1 {
			c.Client.StoreLastDUID(models.LastDevEUI{ShortCode: "<something invalid>"})
		}
		t.Run(fmt.Sprintf("#%d: %q", i, test.testName), func(t *testing.T) {
			uid, err := generateDUID(c.Client)
			if test.err != "" {
				assert.Error(t, err, test.err)
				assert.DeepEqual(t, uid, models.DevEUI{})
			} else {
				assert.Equal(t, reflect.TypeOf(uid), reflect.TypeOf(models.DevEUI{}))
				assert.Equal(t, uid.DevEUI != "", true)
				assert.Equal(t, uid.ShortCode != "", true)
				assert.Equal(t, uid.ShortCode, test.shortcode)
				assert.Equal(t, len(uid.DevEUI), 16)
				assert.Equal(t, len(uid.ShortCode), 5)
			}

		})
	}
}
func TestGenerateBatchValues(t *testing.T) {
	dataType := &[]*models.DevEUI{}
	c := cache.Cache{}
	c.Initialise("", false)

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

// 5 digit HEX code generation
// maximum possible unique device lookups
// 1048576 == (16^5)
//
// read hexcode of previous count - prevents database full table scan
// ensure correct hex format -
// - regex ^[a-fA-F0-9]{1,5}$
//		match between 1-5 chars within 'hex' range
// check for overflow errors
// hex math to range to 100
// concurrently increment and store next value
// build devEUI struct list

func TestCreateHexShortcode(t *testing.T) {
	t.Run("GENERATE 5 digit hex code", func(t *testing.T) {
		suite := []struct {
			testName string
			lastID   string
			expect   string
			err      string
		}{
			{"GENERATE - ", "00000", "00001", ""},
			{"GENERATE - ", "1234F", "12350", ""},
			{"GENERATE - ", "AFD1G", "", "invalid hexcode supplied"},
			{"GENERATE - ", "FFFFF", "", "maximum possible values reached"},
			{"GENERATE - ", "-1FFFE", "", "invalid hexcode supplied"},
		}

		for i, test := range suite {
			t.Run(fmt.Sprintf("#%d - %q: %q", i, test.testName, test.lastID), func(t *testing.T) {
				newHEX, err := generateHexID(test.lastID)
				assert.Equal(t, newHEX, test.expect)
				if test.err == "" {
					assert.NilError(t, err)
				} else {
					assert.Error(t, err, test.err)
				}

			})
		}

	})

}

// Ensure operation is tested
func BenchmarkGenerateHex(b *testing.B) {
	b.Run("GENERATE HEX", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = generateHexID("aabaa")
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
