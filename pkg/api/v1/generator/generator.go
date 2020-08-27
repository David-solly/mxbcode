package generator

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/David-solly/mxbcode/pkg/api/v1/cache"
	"github.com/David-solly/mxbcode/pkg/api/v1/models"
)

// Maximum hex limit corresponding to
// 5 digits of 16 possible values each
// fffff - in decimal form
const shortcodeLimit = 1048575

// Maximum hex limit corresponding to
// 11 digits of 16 possible values each
// fffffffffff - in decimal form
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

// random generator
//
func generateDUID(c cache.Service) (models.DevEUI, error) {
	last, k, err := c.ReadCache(cache.LastUIDKey)
	if err != nil {
		return models.DevEUI{}, err
	}
	if !k {
		return models.DevEUI{}, errors.New("Could not retrieve last unique ID")

	}
	hID, err := generateHexID(last)
	if err != nil {
		return models.DevEUI{}, err
	}
	rand.Seed(time.Now().UnixNano())
	dec := rand.Int63n(trunkBarcodeLimit)
	c.StoreLastDUID(models.LastDevEUI{ShortCode: hID})
	return models.DevEUI{
		ShortCode: hID,
		DevEUI:    fmt.Sprintf("%011s%s", strconv.FormatInt(dec, 16), hID),
	}, nil
}

func generateHexID(lastID string) (string, error) {
	if k, err := parseHex(lastID); k == -1 {
		return "", err
	}

	if strings.ToUpper(lastID) == "FFFFF" {
		return "", fmt.Errorf("maximum possible values reached")
	}

	dec, _ := strconv.ParseInt(lastID, 16, 64)

	return fmt.Sprintf("%05s", strconv.FormatInt(dec+1, 16)), nil
}

func parseHex(hex string) (int64, error) {
	validHex, _ := regexp.MatchString(`^[a-fA-F0-9]{1,5}$`, hex)
	if !validHex {
		return -1, fmt.Errorf("invalid hexcode supplied %q", hex)
	}
	dec, _ := strconv.ParseInt(hex, 16, 64)

	return dec, nil
}

// GenerateDUIDBatch :
// Generae `count` uid's and stores them in `c` when done
//
func GenerateDUIDBatch(count int, c cache.Service) (*[]*models.DevEUI, error) {
	if count < 1 {
		return nil, errors.New("Minimum request is 1")
	}

	if count > DefaultMaxToGenerate {
		return nil, fmt.Errorf("Too many requested - Maximum %d", DefaultMaxToGenerate)
	}

	last, _, err := c.ReadCache(cache.LastUIDKey)
	if err != nil {
		return nil, err
	}

	start, err := parseHex(last)
	if err != nil {
		return nil, err
	}

	batchShorts := start + int64(count)
	if batchShorts > shortcodeLimit {
		return nil, fmt.Errorf("insufficient ID space (%d) remaining to generate (%d) IDs", (shortcodeLimit - start), count)
	}

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

func generateBarcodeTrunk(data *models.DevEUI) {
	dec := rand.Int63n(trunkBarcodeLimit)
	data.DevEUI = fmt.Sprintf("%011s%s", strconv.FormatInt(dec, 16), data.ShortCode)

}
