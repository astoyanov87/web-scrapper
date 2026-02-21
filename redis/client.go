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

// ClearAppCache removes application-specific keys from Redis without
// relying on FLUSHDB/FLUSHALL. It scans for keys with known prefixes
// (e.g. "match:", "player:") and deletes them in pipelined batches.
func ClearAppCache() error {
	if Rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}

	// patterns to remove (prefix-based)
	patterns := []string{"match:*", "player:*"}

	for _, pattern := range patterns {
		var cursor uint64
		for {
			keys, next, err := Rdb.Scan(cursor, pattern, 100).Result()
			if err != nil {
				return fmt.Errorf("scan failed for pattern %s: %v", pattern, err)
			}
			cursor = next

			if len(keys) > 0 {
				pipe := Rdb.Pipeline()
				for _, k := range keys {
					pipe.Del(k)
				}
				if _, err := pipe.Exec(); err != nil {
					return fmt.Errorf("failed to delete keys for pattern %s: %v", pattern, err)
				}
			}

			if cursor == 0 {
				break
			}
		}
	}

	// delete well-known set keys
	wellKnown := []string{"live_matches", "completed_matches", "scheduled_matches"}
	if err := Rdb.Del(wellKnown...).Err(); err != nil {
		return fmt.Errorf("failed to delete well-known keys: %v", err)
	}

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
