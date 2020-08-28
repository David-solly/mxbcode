package generator

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/David-solly/mxbcode/pkg/cache"
	"github.com/David-solly/mxbcode/pkg/models"
)

// Maximum hex limit corresponding to
// 5 digits of 16 possible values each.
// fffff - in decimal form
// This is the actual ID space of all the DevEUIs
// its not possible to have more 1048575 unique
// values using only 5 HEX digits
const shortcodeLimit = 1048575

// Maximum hex limit corresponding to
// 11 digits of 16 possible values each
// fffffffffff - in decimal form
// for this exercise - they might as well all be the same
// since the ID space is limited to the last 5 digits - the shortcode
// whose range is far below this one.
const trunkBarcodeLimit = 17592186044415

// DefaultMaxToGenerate : default number of ids to generate
// set the global item to alter
// can be extended or reduced
const DefaultMaxToGenerate = 100

// DefaultMaxToRegister : default number of ids to register
// with LoRaWAN provider
// set the global item to alter
// can be extended or reduced
const DefaultMaxToRegister = 100

// GenerateDUIDBatch :
// Generate `count` uid's and stores them in `c` when done
func GenerateDUIDBatch(count int, c cache.Service) (*[]*models.DevEUI, error) {
	if count < 1 {
		return nil, errors.New("Minimum request is 1")
	}

	if count > DefaultMaxToGenerate {
		return nil, fmt.Errorf("Too many requested - Maximum %d", DefaultMaxToGenerate)
	}

	// read last incremented hexcode of previous operation
	// - prevents database full table scan
	last, _, err := c.ReadCache(cache.LastUIDKey)
	if err != nil {
		return nil, err
	}

	// ensure hex value matches our criteria
	start, err := parseHex(last)
	if err != nil {
		return nil, err
	}

	// check that the desired range is within physical limits
	// 5 digit HEX code generation
	// maximum possible unique device lookups
	// 1048576 == (16^5)
	//
	// check for overflow errors
	// hex math to range to 100
	batchShorts := start + int64(count)
	if batchShorts > shortcodeLimit {
		return nil, fmt.Errorf("insufficient ID space (%d) remaining to generate (%d) IDs", (shortcodeLimit - start), count)
	}

	// build devEUI struct list
	ids := make([]*models.DevEUI, count)
	rand.Seed(time.Now().UnixNano())
	for i := range ids {
		v := models.DevEUI{ShortCode: fmt.Sprintf("%05s", strconv.FormatInt(start+int64(i+1), 16))}
		generateBarcodeTrunk(&v)
		c.StoreLastDUID(models.LastDevEUI{ShortCode: v.ShortCode})
		ids[i] = &v

	}

	return &ids, nil
}

// ensure hex value matches our criteria
// ensure correct hex format -
// - regex ^[a-fA-F0-9]{1,5}$
//	 match between 1-5 chars within 'hex' range
func parseHex(hex string) (int64, error) {
	validHex, _ := regexp.MatchString(`^[a-fA-F0-9]{1,5}$`, hex)
	if !validHex {
		return -1, fmt.Errorf("invalid hexcode supplied %q", hex)
	}
	// convert hex code to int to work with the math easier
	return strconv.ParseInt(hex, 16, 64)
}

// random generation of the other 11 digits -
// for this excercise - these 11 hex digits are non consequential
// since there is a clause of using a 5 digit unique key, the other 11 digits
// essentially are of no consequence - they are there for aesthetics in this case.
// as long as the 5 shortcode digits are unique, the whole 16 will be unique
// as long as the 5 are shortcode is not unique - the whole 16 is of no use
// rendering the remaining 11 digits of no real consequence in this case
func generateBarcodeTrunk(data *models.DevEUI) {
	dec := rand.Int63n(trunkBarcodeLimit)
	data.DevEUI = fmt.Sprintf("%011s%s", strconv.FormatInt(dec, 16), data.ShortCode)
}
