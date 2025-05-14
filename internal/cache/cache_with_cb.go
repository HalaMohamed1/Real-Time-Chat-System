package cache

import (
	"context"
	"fmt"
	"log"
	"rtcs/internal/circuitbreaker"
	"rtcs/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisWithCircuitBreaker struct {
	redis            *RedisCache
	cbRegistry       *circuitbreaker.Registry
	operationTimeout time.Duration
}

func NewRedisWithCircuitBreaker(client *redis.Client, cbRegistry *circuitbreaker.Registry) *RedisWithCircuitBreaker {
	return &RedisWithCircuitBreaker{
		redis:            NewRedisCache(client),
		cbRegistry:       cbRegistry,
		operationTimeout: 1 * time.Second,
	}
}

func (c *RedisWithCircuitBreaker) SetMessage(ctx context.Context, message *model.Message) error {
	cb := c.cbRegistry.Get("redis-set-message",
		circuitbreaker.WithFailureThreshold(5),
		circuitbreaker.WithResetTimeout(10*time.Second),
		circuitbreaker.WithHalfOpenMaxCalls(2))

	cb = circuitbreaker.LoggingMiddleware(cb)

	return cb.Execute(func() error {
		ctx, cancel := context.WithTimeout(ctx, c.operationTimeout)
		defer cancel()

		err := c.redis.SetMessage(ctx, message)
		if err != nil {
			log.Printf("[ERROR] Redis SetMessage failed: %v", err)
		}
		return err
	})
}

func (c *RedisWithCircuitBreaker) GetMessage(ctx context.Context, messageID string) (*model.Message, error) {
	cb := c.cbRegistry.Get("redis-get-message")

	var message *model.Message
	err := cb.Execute(func() error {
		ctx, cancel := context.WithTimeout(ctx, c.operationTimeout)
		defer cancel()

		var err error
		message, err = c.redis.GetMessage(ctx, messageID)
		if err != nil && err != redis.Nil {
			log.Printf("[ERROR] Redis GetMessage failed: %v", err)
			return err
		}
		return nil
	})

	if err == circuitbreaker.ErrCircuitOpen {
		log.Printf("[CIRCUIT BREAKER] Circuit is open for Redis GetMessage, falling back to default behavior")
		return nil, fmt.Errorf("service temporarily unavailable: %w", err)
	}

	return message, err
}

func (c *RedisWithCircuitBreaker) DeleteMessage(ctx context.Context, messageID string) error {
	cb := c.cbRegistry.Get("redis-delete-message")

	return cb.Execute(func() error {
		ctx, cancel := context.WithTimeout(ctx, c.operationTimeout)
		defer cancel()

		err := c.redis.DeleteMessage(ctx, messageID)
		if err != nil {
			log.Printf("[ERROR] Redis DeleteMessage failed: %v", err)
		}
		return err
	})
}

func (c *RedisWithCircuitBreaker) SetChatMessages(ctx context.Context, chatID string, messages []*model.Message) error {
	cb := c.cbRegistry.Get("redis-set-chat-messages")

	return cb.Execute(func() error {
		ctx, cancel := context.WithTimeout(ctx, c.operationTimeout)
		defer cancel()

		err := c.redis.SetChatMessages(ctx, chatID, messages)
		if err != nil {
			log.Printf("[ERROR] Redis SetChatMessages failed: %v", err)
		}
		return err
	})
}

func (c *RedisWithCircuitBreaker) GetChatMessages(ctx context.Context, chatID string) ([]*model.Message, error) {
	cb := c.cbRegistry.Get("redis-get-chat-messages")

	var messages []*model.Message
	err := cb.Execute(func() error {
		ctx, cancel := context.WithTimeout(ctx, c.operationTimeout)
		defer cancel()

		var err error
		messages, err = c.redis.GetChatMessages(ctx, chatID)
		if err != nil && err != redis.Nil {
			log.Printf("[ERROR] Redis GetChatMessages failed: %v", err)
			return err
		}
		return nil
	})

	if err == circuitbreaker.ErrCircuitOpen {
		log.Printf("[CIRCUIT BREAKER] Circuit is open for Redis GetChatMessages, falling back to default behavior")
		return nil, fmt.Errorf("service temporarily unavailable: %w", err)
	}

	return messages, err
}
