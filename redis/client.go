package redis

import (
	"log"

	"github.com/go-redis/redis"
)

var Rdb *redis.Client

// InitRedis initializes a Redis client
func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr: "192.168.100.254:6379",
		DB:   0,
	})

	// Ping Redis to check the connection
	_, err := Rdb.Ping().Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	log.Println("Connected to Redis")
}
