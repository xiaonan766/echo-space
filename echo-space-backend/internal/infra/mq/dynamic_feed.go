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

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const (
	defaultDynamicFeedQueue  = "echo-space.dynamic.feed"
	dynamicFeedHandleTimeout = 30 * time.Second
)

type DynamicFeedPublisher struct {
	client    *RabbitClient
	queueName string
}

func NewDynamicFeedPublisher(client *RabbitClient, queueName string) *DynamicFeedPublisher {
	return &DynamicFeedPublisher{client: client, queueName: normalizeDynamicFeedQueueName(queueName)}
}

func (p *DynamicFeedPublisher) PublishDynamicFeedMessage(ctx context.Context, message domain.DynamicFeedMessage) error {
	if p == nil || p.client == nil {
		return errors.New("dynamic feed publisher is nil")
	}
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal dynamic feed message: %w", err)
	}
	return publishPersistentWithConfirm(ctx, p.client, p.queueName, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    message.MessageID,
		Timestamp:    time.Now(),
		Body:         body,
	})
}

type DynamicFeedHandler interface {
	HandleDynamicFeedMessage(ctx context.Context, message domain.DynamicFeedMessage) error
}

type DynamicFeedConsumer struct {
	client        *RabbitClient
	queueName     string
	prefetchCount int
	handler       DynamicFeedHandler

	channel *amqp.Channel
	stopCh  chan struct{}
	mu      sync.Mutex
	once    sync.Once
}

func NewDynamicFeedConsumer(client *RabbitClient, queueName string, prefetchCount int, handler DynamicFeedHandler) *DynamicFeedConsumer {
	if prefetchCount <= 0 {
		prefetchCount = 20
	}
	return &DynamicFeedConsumer{
		client: client, queueName: normalizeDynamicFeedQueueName(queueName),
		prefetchCount: prefetchCount, handler: handler, stopCh: make(chan struct{}),
	}
}

func (c *DynamicFeedConsumer) Start(ctx context.Context) error {
	if c == nil || c.client == nil || c.handler == nil {
		return errors.New("dynamic feed consumer is not ready")
	}
	go c.run(ctx)
	return nil
}

func (c *DynamicFeedConsumer) run(ctx context.Context) {
	for {
		if err := c.consumeOnce(ctx); err != nil {
			if isConsumerStopped(ctx, c.stopCh) {
				return
			}
			log.Printf("dynamic feed consumer stopped, will restart: queue=%s err=%v", c.queueName, err)
		}
		if !waitConsumerRetry(ctx, c.stopCh) {
			return
		}
	}
}

func (c *DynamicFeedConsumer) consumeOnce(ctx context.Context) error {
	channel, err := c.client.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()
	if err := declareDurableQueue(channel, c.queueName); err != nil {
		return err
	}
	if err := channel.Qos(c.prefetchCount, 0, false); err != nil {
		return fmt.Errorf("set dynamic feed qos: %w", err)
	}
	closeCh := channel.NotifyClose(make(chan *amqp.Error, 1))
	deliveries, err := channel.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume dynamic feed queue: %w", err)
	}

	c.setChannel(channel)
	defer c.clearChannel(channel)
	for {
		select {
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("dynamic feed delivery channel closed")
			}
			c.handleDelivery(delivery)
		case closeErr := <-closeCh:
			if closeErr != nil {
				return fmt.Errorf("dynamic feed channel closed: %w", closeErr)
			}
			return errors.New("dynamic feed channel closed")
		case <-c.stopCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *DynamicFeedConsumer) handleDelivery(delivery amqp.Delivery) {
	var message domain.DynamicFeedMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil || strings.TrimSpace(message.MessageID) == "" {
		log.Printf("discard invalid dynamic feed message: %v", err)
		_ = delivery.Ack(false)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), dynamicFeedHandleTimeout)
	defer cancel()
	if err := c.handler.HandleDynamicFeedMessage(ctx, message); err != nil {
		log.Printf("handle dynamic feed message failed: messageID=%s videoID=%s err=%v", message.MessageID, message.VideoID, err)
		_ = delivery.Nack(false, true)
		return
	}
	_ = delivery.Ack(false)
}

func (c *DynamicFeedConsumer) Close() {
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

func (c *DynamicFeedConsumer) setChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channel = channel
}

func (c *DynamicFeedConsumer) clearChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channel == channel {
		c.channel = nil
	}
}

func normalizeDynamicFeedQueueName(queueName string) string {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return defaultDynamicFeedQueue
	}
	return queueName
}
