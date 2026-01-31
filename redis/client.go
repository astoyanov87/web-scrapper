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

// DeleteKeysByPattern deletes all keys matching the given pattern using SCAN
func DeleteKeysByPattern(pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := Rdb.Scan(cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := Rdb.Del(keys...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// ClearAppCache clears all application-specific keys (match:* and tournamentId)
func ClearAppCache() error {
	if err := DeleteKeysByPattern("match:*"); err != nil {
		return fmt.Errorf("failed to delete match keys: %v", err)
	}
	if err := Rdb.Del("tournamentId").Err(); err != nil {
		return fmt.Errorf("failed to delete tournamentId: %v", err)
	}
	return nil
}
