package redis

import (
	"fmt"
	"log"

	"github.com/astoyanov87/web-scrapper/config"
	"github.com/go-redis/redis"
)

var Rdb *redis.Client

// InitRedis initializes a Redis client using the provided configuration
func InitRedis(cfg *config.Config) error {
	redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	
	Rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       0,
	})

	// Ping Redis to check the connection
	if _, err := Rdb.Ping().Result(); err != nil {
		return fmt.Errorf("could not connect to Redis at %s: %v", redisAddr, err)
	}

	log.Printf("Connected to Redis at %s", redisAddr)
	return nil
}
