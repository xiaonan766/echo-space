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
	rabbitReconnectDelay      = 3 * time.Second
)

type RabbitClient struct {
	url    string
	conn   *amqp.Connection
	mu     sync.Mutex
	closed bool
}

func NewRabbitClient(_ context.Context, cfg config.RabbitMQConfig) (*RabbitClient, error) {
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		return nil, errors.New("rabbitmq url is empty")
	}

	client := &RabbitClient{url: url}
	if err := client.reconnectLocked(); err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}
	return client, nil
}

func (c *RabbitClient) Channel() (*amqp.Channel, error) {
	if c == nil {
		return nil, errors.New("rabbitmq client is nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, errors.New("rabbitmq client is closed")
	}
	if c.conn == nil || c.conn.IsClosed() {
		if err := c.reconnectLocked(); err != nil {
			return nil, err
		}
	}

	channel, err := c.conn.Channel()
	if err == nil {
		return channel, nil
	}
	if !isRabbitReconnectableError(err) {
		return nil, err
	}

	c.closeLocked()
	if err := c.reconnectLocked(); err != nil {
		return nil, err
	}
	return c.conn.Channel()
}

func (c *RabbitClient) Reconnect() error {
	if c == nil {
		return errors.New("rabbitmq client is nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("rabbitmq client is closed")
	}
	c.closeLocked()
	return c.reconnectLocked()
}

func (c *RabbitClient) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	c.closeLocked()
}

func (c *RabbitClient) reconnectLocked() error {
	conn, err := amqp.DialConfig(c.url, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	c.conn = conn
	return nil
}

func (c *RabbitClient) closeLocked() {
	if c.conn != nil && !c.conn.IsClosed() {
		_ = c.conn.Close()
	}
	c.conn = nil
}

func publishPersistentWithConfirm(ctx context.Context, client *RabbitClient, queueName string, publishing amqp.Publishing) error {
	if client == nil {
		return errors.New("rabbitmq client is nil")
	}

	publishCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		err := publishPersistentOnce(publishCtx, client, queueName, publishing)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRabbitReconnectableError(err) || publishCtx.Err() != nil {
			return err
		}
		if reconnectErr := client.Reconnect(); reconnectErr != nil {
			return fmt.Errorf("%w; reconnect rabbitmq failed: %v", err, reconnectErr)
		}
	}
	return lastErr
}

func publishPersistentOnce(ctx context.Context, client *RabbitClient, queueName string, publishing amqp.Publishing) error {
	channel, err := client.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := declareDurableQueue(channel, queueName); err != nil {
		return err
	}
	if err := channel.Confirm(false); err != nil {
		return fmt.Errorf("enable rabbitmq confirm: %w", err)
	}
	confirmCh := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	if err := channel.PublishWithContext(ctx, "", queueName, false, false, publishing); err != nil {
		return fmt.Errorf("publish rabbitmq message: %w", err)
	}

	select {
	case confirm, ok := <-confirmCh:
		if !ok {
			return errors.New("rabbitmq publish confirm channel closed")
		}
		if !confirm.Ack {
			return errors.New("rabbitmq did not ack message")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func isRabbitReconnectableError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	reconnectablePatterns := []string{
		"connection is not open",
		"channel/connection is not open",
		"channel is not open",
		"connection closed",
		"channel closed",
		"confirm channel closed",
		"exception (320)",
		"exception (504)",
		"broken pipe",
		"connection reset",
		"eof",
	}
	for _, pattern := range reconnectablePatterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}
	return false
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

	return publishPersistentWithConfirm(
		ctx,
		p.client,
		p.queueName,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
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
	mu      sync.Mutex
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

func (c *ShopCacheRecoveryConsumer) Start(ctx context.Context) error {
	if c == nil || c.client == nil || c.handler == nil {
		return errors.New("shop cache recovery consumer is not ready")
	}
	go c.run(ctx)
	return nil
}

func (c *ShopCacheRecoveryConsumer) run(ctx context.Context) {
	for {
		if err := c.consumeOnce(ctx); err != nil {
			if isConsumerStopped(ctx, c.stopCh) {
				return
			}
			log.Printf("shop cache recovery consumer stopped, will restart: queue=%s err=%v", c.queueName, err)
		}
		if !waitConsumerRetry(ctx, c.stopCh) {
			return
		}
	}
}

func (c *ShopCacheRecoveryConsumer) consumeOnce(ctx context.Context) error {
	channel, err := c.client.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := declareDurableQueue(channel, c.queueName); err != nil {
		return err
	}
	if err := channel.Qos(c.prefetchCount, 0, false); err != nil {
		return fmt.Errorf("set rabbitmq qos: %w", err)
	}
	closeCh := channel.NotifyClose(make(chan *amqp.Error, 1))

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
		return fmt.Errorf("consume shop cache recovery queue: %w", err)
	}

	c.setChannel(channel)
	defer c.clearChannel(channel)

	for {
		select {
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("shop cache recovery delivery channel closed")
			}
			c.handleDelivery(delivery)
		case closeErr := <-closeCh:
			if closeErr != nil {
				return fmt.Errorf("shop cache recovery channel closed: %w", closeErr)
			}
			return errors.New("shop cache recovery channel closed")
		case <-c.stopCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *ShopCacheRecoveryConsumer) Close() {
	if c == nil {
		return
	}
	c.once.Do(func() {
		close(c.stopCh)
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.channel != nil {
			_ = c.channel.Close()
			c.channel = nil
		}
	})
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

func (c *ShopCacheRecoveryConsumer) setChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channel = channel
}

func (c *ShopCacheRecoveryConsumer) clearChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channel == channel {
		c.channel = nil
	}
}

func isConsumerStopped(ctx context.Context, stopCh <-chan struct{}) bool {
	select {
	case <-stopCh:
		return true
	default:
	}
	return ctx.Err() != nil
}

func waitConsumerRetry(ctx context.Context, stopCh <-chan struct{}) bool {
	timer := time.NewTimer(rabbitReconnectDelay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-stopCh:
		return false
	case <-ctx.Done():
		return false
	}
}

func normalizeQueueName(queueName string) string {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return defaultCacheRecoveryQueue
	}
	return queueName
}
