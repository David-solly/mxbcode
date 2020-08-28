package cache

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/David-solly/mxbcode/pkg/models"

	"github.com/go-redis/redis"
)

type RedisCache struct {
	client *redis.Client
}

// Init : redis
func (c *RedisCache) init() (string, error) {
	// From Deployment or environmental variables
	dsn := os.Getenv("REDIS_DSN")
	if len(dsn) == 0 {
		return "", errors.New("No address supplied")
	}
	c.client = redis.NewClient(&redis.Options{
		Addr: dsn, //redis port
	})
	k, err := c.client.Ping().Result()
	if err != nil {
		return k, err
	}
	resp := c.client.Get(LastUIDKey)
	str, err := resp.Result()

	// Verify the hexcode
	validHex, _ := regexp.MatchString(`^[a-fA-F0-9]{1,5}$`, str)
	if err != nil || !validHex {
		base := c.client.Set(LastUIDKey, "00000", 0)
		errAccess := base.Err()
		if errAccess != nil {
			return "", errAccess

		}
	}

	fmt.Println("Redis server - Online ..........")
	return k, nil
}

func (c *RedisCache) Initialise() (string, error) {
	return c.init()

}

func (c *RedisCache) StoreDUID(model models.DevEUI) (bool, error) {
	if c.client == nil {
		return false, errors.New("Redis client is nil")
	}
	base := c.client.Set(strings.ToUpper(model.ShortCode), strings.ToUpper(model.DevEUI), 0)
	errAccess := base.Err()
	if errAccess != nil {
		return false, errAccess
	}
	return true, nil
}

func (c *RedisCache) StoreLastDUID(model models.LastDevEUI) (bool, error) {
	base := c.client.Set(LastUIDKey, strings.ToUpper(model.ShortCode), 0)
	errAccess := base.Err()
	if errAccess != nil {
		return false, errAccess
	}
	return true, nil
}

// StoreDUIDGenResponse :
//For caching generate results from the same client - idempotent cache store
//
func (c *RedisCache) StoreDUIDGenResponse(model models.ApiResponseCacheObject) (bool, error) {
	base := c.client.Set(strings.ToUpper(model.Key), model.Response, model.Timeout)
	errAccess := base.Err()
	if errAccess != nil {
		return false, errAccess
	}
	return true, nil
}

func (c *RedisCache) ReadCache(key string) (string, bool, error) {
	data, err := c.client.Get(strings.ToUpper(key)).Result()

	if err != nil {
		return "", false, fmt.Errorf("Device id with shortcode: '%q' - Not Found", key)
	}
	return data, true, nil
}
