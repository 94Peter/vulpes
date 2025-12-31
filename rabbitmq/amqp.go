package rabbitmq

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/94peter/vulpes/constant"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	conn    *amqp.Connection
	channel *amqp.Channel

	queues = make(map[string]amqp.Queue) // 存已宣告的 queue
	mu     sync.RWMutex
)

// Init 建立連線與 channel (全域)
func Init(url string) error {
	var err error
	conn, err = amqp.Dial(url)
	if err != nil {
		return err
	}

	channel, err = conn.Channel()
	if err != nil {
		return err
	}
	return nil
}

// Close 關閉全域連線
func Close() {
	if channel != nil {
		_ = channel.Close()
	}
	if conn != nil {
		_ = conn.Close()
	}
}

// DeclareQueue 宣告 queue 並存到 map
func DeclareQueue(name string) (amqp.Queue, error) {
	q, err := channel.QueueDeclare(
		name,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return q, err
	}

	mu.Lock()
	queues[name] = q
	mu.Unlock()
	return q, nil
}

// Send 發送訊息 (自動查 map)
func Send(queue string, body string) error {
	mu.RLock()
	_, ok := queues[queue]
	mu.RUnlock()
	if !ok {
		return errors.New("queue not declared: " + queue)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constant.DefaultTimeout)
	defer cancel()

	return channel.PublishWithContext(ctx,
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
}

// Consume 消費訊息
func Consume(queue string, handler func(msg string) error) error {
	mu.RLock()
	_, ok := queues[queue]
	mu.RUnlock()
	if !ok {
		return errors.New("queue not declared: " + queue)
	}

	msgs, err := channel.Consume(
		queue,
		"",
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			if err := handler(string(d.Body)); err != nil {
				log.Printf("Handler error: %v", err)
				// 重試一次
				_ = d.Nack(false, true) // true 表示 requeue
			} else {
				_ = d.Ack(false)
			}
		}
	}()
	return nil
}
