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
	defaultVideoHotMetricQueue  = "echo-space.video.hot.metric"
	videoHotMetricHandleTimeout = 10 * time.Second
)

type VideoHotMetricPublisher struct {
	client    *RabbitClient
	queueName string
}

func NewVideoHotMetricPublisher(client *RabbitClient, queueName string) *VideoHotMetricPublisher {
	return &VideoHotMetricPublisher{
		client:    client,
		queueName: normalizeVideoHotMetricQueueName(queueName),
	}
}

func (p *VideoHotMetricPublisher) PublishVideoHotMetricEvent(ctx context.Context, event domain.VideoHotMetricEvent) error {
	if p == nil || p.client == nil {
		return errors.New("video hot metric publisher is nil")
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal video hot metric event: %w", err)
	}
	return publishPersistentWithConfirm(ctx, p.client, p.queueName, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    event.EventID,
		Timestamp:    time.Now(),
		Body:         body,
	})
}

type VideoHotMetricHandler interface {
	HandleVideoHotMetricEvent(ctx context.Context, event domain.VideoHotMetricEvent) error
}

type VideoHotMetricConsumer struct {
	client        *RabbitClient
	queueName     string
	prefetchCount int
	handler       VideoHotMetricHandler

	channel *amqp.Channel
	stopCh  chan struct{}
	mu      sync.Mutex
	once    sync.Once
}

func NewVideoHotMetricConsumer(client *RabbitClient, queueName string, prefetchCount int, handler VideoHotMetricHandler) *VideoHotMetricConsumer {
	if prefetchCount <= 0 {
		prefetchCount = 20
	}
	return &VideoHotMetricConsumer{
		client:        client,
		queueName:     normalizeVideoHotMetricQueueName(queueName),
		prefetchCount: prefetchCount,
		handler:       handler,
		stopCh:        make(chan struct{}),
	}
}

func (c *VideoHotMetricConsumer) Start(ctx context.Context) error {
	if c == nil || c.client == nil || c.handler == nil {
		return errors.New("video hot metric consumer is not ready")
	}
	go c.run(ctx)
	return nil
}

func (c *VideoHotMetricConsumer) run(ctx context.Context) {
	for {
		if err := c.consumeOnce(ctx); err != nil {
			if isConsumerStopped(ctx, c.stopCh) {
				return
			}
			log.Printf("video hot metric consumer stopped, will restart: queue=%s err=%v", c.queueName, err)
		}
		if !waitConsumerRetry(ctx, c.stopCh) {
			return
		}
	}
}

func (c *VideoHotMetricConsumer) consumeOnce(ctx context.Context) error {
	channel, err := c.client.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := declareDurableQueue(channel, c.queueName); err != nil {
		return err
	}
	if err := channel.Qos(c.prefetchCount, 0, false); err != nil {
		return fmt.Errorf("set video hot metric qos: %w", err)
	}
	closeCh := channel.NotifyClose(make(chan *amqp.Error, 1))
	deliveries, err := channel.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume video hot metric queue: %w", err)
	}

	c.setChannel(channel)
	defer c.clearChannel(channel)
	for {
		select {
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("video hot metric delivery channel closed")
			}
			c.handleDelivery(delivery)
		case closeErr := <-closeCh:
			if closeErr != nil {
				return fmt.Errorf("video hot metric channel closed: %w", closeErr)
			}
			return errors.New("video hot metric channel closed")
		case <-c.stopCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *VideoHotMetricConsumer) handleDelivery(delivery amqp.Delivery) {
	var event domain.VideoHotMetricEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil || strings.TrimSpace(event.EventID) == "" {
		log.Printf("discard invalid video hot metric event: %v", err)
		_ = delivery.Ack(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), videoHotMetricHandleTimeout)
	defer cancel()
	if err := c.handler.HandleVideoHotMetricEvent(ctx, event); err != nil {
		log.Printf("handle video hot metric event failed: eventID=%s videoID=%s eventType=%s err=%v", event.EventID, event.VideoID, event.EventType, err)
		_ = delivery.Nack(false, true)
		return
	}
	_ = delivery.Ack(false)
}

func (c *VideoHotMetricConsumer) Close() {
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

func (c *VideoHotMetricConsumer) setChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channel = channel
}

func (c *VideoHotMetricConsumer) clearChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channel == channel {
		c.channel = nil
	}
}

func normalizeVideoHotMetricQueueName(queueName string) string {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return defaultVideoHotMetricQueue
	}
	return queueName
}
