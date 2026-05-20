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
			MessageId:    message.MessageID,
			Timestamp:    time.Now(),
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish shop stock lock message: %w", err)
	}

	select {
	case confirm := <-confirmCh:
		if !confirm.Ack {
			return errors.New("rabbitmq did not ack shop stock lock message")
		}
		return nil
	case <-publishCtx.Done():
		return publishCtx.Err()
	}
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

func (c *ShopStockLockConsumer) Start(_ context.Context) error {
	if c == nil || c.client == nil || c.handler == nil {
		return errors.New("shop stock lock consumer is not ready")
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
		return fmt.Errorf("consume shop stock lock queue: %w", err)
	}

	c.channel = channel
	go c.consume(deliveries)
	return nil
}

func (c *ShopStockLockConsumer) Close() {
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

func (c *ShopStockLockConsumer) consume(deliveries <-chan amqp.Delivery) {
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

func normalizeStockLockQueueName(queueName string) string {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return defaultStockLockQueue
	}
	return queueName
}
