package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
)

const (
	defaultCacheRecoveryQueue = "echo-space.shop.cache.recovery"
	publishTimeout            = 3 * time.Second
	consumeHandleTimeout      = 10 * time.Second
)

type RabbitClient struct {
	conn *amqp.Connection
	once sync.Once
}

func NewRabbitClient(_ context.Context, cfg config.RabbitMQConfig) (*RabbitClient, error) {
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		return nil, errors.New("rabbitmq url is empty")
	}

	conn, err := amqp.DialConfig(url, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}
	return &RabbitClient{conn: conn}, nil
}

func (c *RabbitClient) Channel() (*amqp.Channel, error) {
	if c == nil || c.conn == nil {
		return nil, errors.New("rabbitmq client is nil")
	}
	return c.conn.Channel()
}

func (c *RabbitClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	c.once.Do(func() {
		_ = c.conn.Close()
	})
}

type ShopCacheRecoveryPublisher struct {
	client    *RabbitClient
	queueName string
}

func NewShopCacheRecoveryPublisher(client *RabbitClient, queueName string) *ShopCacheRecoveryPublisher {
	return &ShopCacheRecoveryPublisher{
		client:    client,
		queueName: normalizeQueueName(queueName),
	}
}

func (p *ShopCacheRecoveryPublisher) PublishShopCacheRecoveryTask(ctx context.Context, task cache.ShopCacheRecoveryTask) error {
	if p == nil || p.client == nil {
		return errors.New("shop cache recovery publisher is nil")
	}

	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal shop cache recovery task: %w", err)
	}

	publishCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	channel, err := p.client.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := declareDurableQueue(channel, p.queueName); err != nil {
		return err
	}

	if err := channel.Confirm(false); err != nil {
		return fmt.Errorf("enable rabbitmq confirm: %w", err)
	}
	confirmCh := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	if err := channel.PublishWithContext(
		publishCtx,
		"",
		p.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish shop cache recovery task: %w", err)
	}

	select {
	case confirm := <-confirmCh:
		if !confirm.Ack {
			return errors.New("rabbitmq did not ack shop cache recovery task")
		}
		return nil
	case <-publishCtx.Done():
		return publishCtx.Err()
	}
}

type ShopCacheRecoveryTaskHandler interface {
	HandleShopCacheRecoveryTask(ctx context.Context, task cache.ShopCacheRecoveryTask) error
}

type ShopCacheRecoveryConsumer struct {
	client        *RabbitClient
	queueName     string
	prefetchCount int
	handler       ShopCacheRecoveryTaskHandler

	channel *amqp.Channel
	stopCh  chan struct{}
	once    sync.Once
}

func NewShopCacheRecoveryConsumer(client *RabbitClient, queueName string, prefetchCount int, handler ShopCacheRecoveryTaskHandler) *ShopCacheRecoveryConsumer {
	if prefetchCount <= 0 {
		prefetchCount = 20
	}
	return &ShopCacheRecoveryConsumer{
		client:        client,
		queueName:     normalizeQueueName(queueName),
		prefetchCount: prefetchCount,
		handler:       handler,
		stopCh:        make(chan struct{}),
	}
}

func (c *ShopCacheRecoveryConsumer) Start(_ context.Context) error {
	if c == nil || c.client == nil || c.handler == nil {
		return errors.New("shop cache recovery consumer is not ready")
	}

	channel, err := c.client.Channel()
	if err != nil {
		return err
	}
	if err := declareDurableQueue(channel, c.queueName); err != nil {
		_ = channel.Close()
		return err
	}
	if err := channel.Qos(c.prefetchCount, 0, false); err != nil {
		_ = channel.Close()
		return fmt.Errorf("set rabbitmq qos: %w", err)
	}

	deliveries, err := channel.Consume(
		c.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = channel.Close()
		return fmt.Errorf("consume shop cache recovery queue: %w", err)
	}

	c.channel = channel
	go c.consume(deliveries)
	return nil
}

func (c *ShopCacheRecoveryConsumer) Close() {
	if c == nil {
		return
	}
	c.once.Do(func() {
		close(c.stopCh)
		if c.channel != nil {
			_ = c.channel.Close()
		}
	})
}

func (c *ShopCacheRecoveryConsumer) consume(deliveries <-chan amqp.Delivery) {
	for {
		select {
		case delivery, ok := <-deliveries:
			if !ok {
				return
			}
			c.handleDelivery(delivery)
		case <-c.stopCh:
			return
		}
	}
}

func (c *ShopCacheRecoveryConsumer) handleDelivery(delivery amqp.Delivery) {
	var task cache.ShopCacheRecoveryTask
	if err := json.Unmarshal(delivery.Body, &task); err != nil {
		log.Printf("discard invalid shop cache recovery task: %v", err)
		_ = delivery.Ack(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), consumeHandleTimeout)
	defer cancel()

	if err := c.handler.HandleShopCacheRecoveryTask(ctx, task); err != nil {
		log.Printf("handle shop cache recovery task failed: type=%s productID=%d key=%s err=%v", task.Type, task.ProductID, task.Key, err)
		_ = delivery.Nack(false, true)
		return
	}
	_ = delivery.Ack(false)
}

func declareDurableQueue(channel *amqp.Channel, queueName string) error {
	_, err := channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare rabbitmq queue %s: %w", queueName, err)
	}
	return nil
}

func normalizeQueueName(queueName string) string {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return defaultCacheRecoveryQueue
	}
	return queueName
}
