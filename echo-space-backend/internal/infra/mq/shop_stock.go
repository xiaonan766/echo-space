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
)

const defaultStockLockQueue = "echo-space.shop.stock.lock"

type ShopStockLockMessage struct {
	MessageID string `json:"messageId"`
	OrderNo   string `json:"orderNo"`
	UserID    string `json:"userId"`
	ProductID uint64 `json:"productId"`
	SkuID     uint64 `json:"skuId"`
	BuyCount  int    `json:"buyCount"`
}

type ShopStockLockPublisher struct {
	client    *RabbitClient
	queueName string
}

func NewShopStockLockPublisher(client *RabbitClient, queueName string) *ShopStockLockPublisher {
	return &ShopStockLockPublisher{
		client:    client,
		queueName: normalizeStockLockQueueName(queueName),
	}
}

func (p *ShopStockLockPublisher) PublishShopStockLockMessage(ctx context.Context, message ShopStockLockMessage) error {
	if p == nil || p.client == nil {
		return errors.New("shop stock lock publisher is nil")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal shop stock lock message: %w", err)
	}

	return publishPersistentWithConfirm(
		ctx,
		p.client,
		p.queueName,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    message.MessageID,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

type ShopStockLockHandler interface {
	HandleShopStockLockMessage(ctx context.Context, message ShopStockLockMessage) error
}

type ShopStockLockConsumer struct {
	client        *RabbitClient
	queueName     string
	prefetchCount int
	handler       ShopStockLockHandler

	channel *amqp.Channel
	stopCh  chan struct{}
	mu      sync.Mutex
	once    sync.Once
}

func NewShopStockLockConsumer(client *RabbitClient, queueName string, prefetchCount int, handler ShopStockLockHandler) *ShopStockLockConsumer {
	if prefetchCount <= 0 {
		prefetchCount = 20
	}
	return &ShopStockLockConsumer{
		client:        client,
		queueName:     normalizeStockLockQueueName(queueName),
		prefetchCount: prefetchCount,
		handler:       handler,
		stopCh:        make(chan struct{}),
	}
}

func (c *ShopStockLockConsumer) Start(ctx context.Context) error {
	if c == nil || c.client == nil || c.handler == nil {
		return errors.New("shop stock lock consumer is not ready")
	}
	go c.run(ctx)
	return nil
}

func (c *ShopStockLockConsumer) run(ctx context.Context) {
	for {
		if err := c.consumeOnce(ctx); err != nil {
			if isConsumerStopped(ctx, c.stopCh) {
				return
			}
			log.Printf("shop stock lock consumer stopped, will restart: queue=%s err=%v", c.queueName, err)
		}
		if !waitConsumerRetry(ctx, c.stopCh) {
			return
		}
	}
}

func (c *ShopStockLockConsumer) consumeOnce(ctx context.Context) error {
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
		return fmt.Errorf("consume shop stock lock queue: %w", err)
	}

	c.setChannel(channel)
	defer c.clearChannel(channel)

	for {
		select {
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("shop stock lock delivery channel closed")
			}
			c.handleDelivery(delivery)
		case closeErr := <-closeCh:
			if closeErr != nil {
				return fmt.Errorf("shop stock lock channel closed: %w", closeErr)
			}
			return errors.New("shop stock lock channel closed")
		case <-c.stopCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *ShopStockLockConsumer) Close() {
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

func (c *ShopStockLockConsumer) handleDelivery(delivery amqp.Delivery) {
	var message ShopStockLockMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		log.Printf("discard invalid shop stock lock message: %v", err)
		_ = delivery.Ack(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), consumeHandleTimeout)
	defer cancel()

	if err := c.handler.HandleShopStockLockMessage(ctx, message); err != nil {
		log.Printf("handle shop stock lock message failed: orderNo=%s skuID=%d err=%v", message.OrderNo, message.SkuID, err)
		_ = delivery.Nack(false, true)
		return
	}
	_ = delivery.Ack(false)
}

func (c *ShopStockLockConsumer) setChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channel = channel
}

func (c *ShopStockLockConsumer) clearChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channel == channel {
		c.channel = nil
	}
}

func normalizeStockLockQueueName(queueName string) string {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return defaultStockLockQueue
	}
	return queueName
}
