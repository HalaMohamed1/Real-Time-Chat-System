package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

// Initialize Redis connection
func InitCache() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password
		DB:       0,                // Default DB
	})

	// Verify connection
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	log.Println("Redis connected")
}

// stores data in Redis with expiration
func SetCache(key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return redisClient.Set(context.Background(), key, jsonData, expiration).Err()
}

// retrieves data from Redis
func GetCache(key string, dest interface{}) error {
	val, err := redisClient.Get(context.Background(), key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// removes data from Redis
func DeleteCache(key string) error {
	return redisClient.Del(context.Background(), key).Err()
}
