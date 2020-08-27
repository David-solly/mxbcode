package cache

import (
	"errors"
	"os"
	"strings"

	"github.com/David-solly/mxbcode/pkg/api/v1/models"
)

// Cache :
// A cache provider
type Cache struct {
	Client Service
}

func (c *Cache) Initialise(redisAddr string, use bool) (bool, error) {
	if use {
		if addr := strings.Trim(redisAddr, " "); addr == "" {
			return false, errors.New("No address supllied")
		}
		os.Setenv("REDIS_DSN", redisAddr)
		c.Client = &RedisCache{}

		// Init the redis client
		pong, err := c.Client.Initialise()
		if err != nil {
			return false, err
		}
		if pong == "PONG" {
			return true, nil

		}

	}
	// Init the memory client
	c.Client = &MemoryCache{}
	pong, _ := c.Client.Initialise()
	if pong == "PONG" {
		return true, nil

	}
	return false, nil
}

// Service :
// A pluggable cache service provider
type Service interface {
	Initialise() (string, error)
	StoreDUID(model models.DevEUI) (bool, error)
	StoreLastDUID(model models.LastDevEUI) (bool, error)
	StoreDUIDGenResponse(model models.ApiResponseCacheObject) (bool, error)
	ReadCache(key string) (string, bool, error)
}
