package swissecho

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type DispatchQueue interface {
	Push(msg *SwissechoMessage) error
	StartWorkers(workers int, dispatchFunc func(msg *SwissechoMessage) (SendResult, error))
}

// MemoryQueue uses standard go channels
type MemoryQueue struct {
	ch chan *SwissechoMessage
}

func NewMemoryQueue() *MemoryQueue {
	return NewMemoryQueueWithSize(1000)
}

// NewMemoryQueueWithSize creates a MemoryQueue with a custom buffer size.
// Useful for testing or tuning throughput.
func NewMemoryQueueWithSize(size int) *MemoryQueue {
	return &MemoryQueue{
		ch: make(chan *SwissechoMessage, size),
	}
}

func (q *MemoryQueue) Push(msg *SwissechoMessage) error {
	select {
	case q.ch <- msg:
		return nil
	default:
		return fmt.Errorf("swissecho: memory queue is full (buffer size %d)", cap(q.ch))
	}
}

func (q *MemoryQueue) StartWorkers(workers int, dispatchFunc func(msg *SwissechoMessage) (SendResult, error)) {
	if workers <= 0 {
		workers = 5
	}
	for i := 0; i < workers; i++ {
		go func() {
			for msg := range q.ch {
				_, err := dispatchFunc(msg)
				if err != nil {
					log.Printf("[Swissecho Async Error] Failed to send message: %v\n", err)
				}
			}
		}()
	}
}

// RedisQueue uses a simple Redis List
type RedisQueue struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisQueue(config RedisConfig) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})
	return &RedisQueue{
		client: client,
		ctx:    context.Background(),
	}
}

func (q *RedisQueue) Push(msg *SwissechoMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return q.client.LPush(q.ctx, "swissecho_queue", b).Err()
}

func (q *RedisQueue) StartWorkers(workers int, dispatchFunc func(msg *SwissechoMessage) (SendResult, error)) {
	if workers <= 0 {
		workers = 5
	}
	for i := 0; i < workers; i++ {
		go func() {
			for {
				result, err := q.client.BRPop(q.ctx, 0, "swissecho_queue").Result()
				if err != nil {
					log.Printf("[Swissecho Redis Error] BRPop failed: %v\n", err)
					time.Sleep(2 * time.Second) // back off before retrying
					continue
				}

				if len(result) == 2 {
					var msg SwissechoMessage
					if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
						log.Printf("[Swissecho Redis Error] Failed to unmarshal message: %v\n", err)
						continue
					}

					_, err := dispatchFunc(&msg)
					if err != nil {
						log.Printf("[Swissecho Async Error] Failed to send message: %v\n", err)
					}
				}
			}
		}()
	}
}
