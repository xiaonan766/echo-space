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

const (
	defaultVideoTranscodeQueue  = "echo-space.video.transcode"
	videoTranscodeHandleTimeout = 3 * time.Hour
)

type VideoTranscodeMessage struct {
	MessageID string `json:"messageId"`
	FileID    string `json:"fileId"`
	VideoID   string `json:"videoId"`
	UserID    string `json:"userId"`
	UploadID  string `json:"uploadId"`
	FilePath  string `json:"filePath"`
	FileName  string `json:"fileName"`
	Chunks    int    `json:"chunks"`
}

type VideoTranscodePublisher struct {
	client    *RabbitClient
	queueName string
}

func NewVideoTranscodePublisher(client *RabbitClient, queueName string) *VideoTranscodePublisher {
	return &VideoTranscodePublisher{client: client, queueName: normalizeVideoTranscodeQueueName(queueName)}
}

func (p *VideoTranscodePublisher) PublishVideoTranscodeMessage(ctx context.Context, message VideoTranscodeMessage) error {
	if p == nil || p.client == nil {
		return errors.New("video transcode publisher is nil")
	}
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal video transcode message: %w", err)
	}
	return publishPersistentWithConfirm(ctx, p.client, p.queueName, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    message.MessageID,
		Timestamp:    time.Now(),
		Body:         body,
	})
}

type VideoTranscodeHandler interface {
	HandleVideoTranscodeMessage(ctx context.Context, message VideoTranscodeMessage) error
}

type VideoTranscodeConsumer struct {
	client        *RabbitClient
	queueName     string
	prefetchCount int
	handler       VideoTranscodeHandler

	channel *amqp.Channel
	stopCh  chan struct{}
	mu      sync.Mutex
	once    sync.Once
}

func NewVideoTranscodeConsumer(client *RabbitClient, queueName string, prefetchCount int, handler VideoTranscodeHandler) *VideoTranscodeConsumer {
	if prefetchCount <= 0 {
		prefetchCount = 1
	}
	return &VideoTranscodeConsumer{
		client: client, queueName: normalizeVideoTranscodeQueueName(queueName),
		prefetchCount: prefetchCount, handler: handler, stopCh: make(chan struct{}),
	}
}

func (c *VideoTranscodeConsumer) Start(ctx context.Context) error {
	if c == nil || c.client == nil || c.handler == nil {
		return errors.New("video transcode consumer is not ready")
	}
	go c.run(ctx)
	return nil
}

func (c *VideoTranscodeConsumer) run(ctx context.Context) {
	for {
		if err := c.consumeOnce(ctx); err != nil {
			if isConsumerStopped(ctx, c.stopCh) {
				return
			}
			log.Printf("video transcode consumer stopped, will restart: queue=%s err=%v", c.queueName, err)
		}
		if !waitConsumerRetry(ctx, c.stopCh) {
			return
		}
	}
}

func (c *VideoTranscodeConsumer) consumeOnce(ctx context.Context) error {
	channel, err := c.client.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()
	if err := declareDurableQueue(channel, c.queueName); err != nil {
		return err
	}
	if err := channel.Qos(c.prefetchCount, 0, false); err != nil {
		return fmt.Errorf("set video transcode qos: %w", err)
	}
	closeCh := channel.NotifyClose(make(chan *amqp.Error, 1))
	deliveries, err := channel.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume video transcode queue: %w", err)
	}

	c.setChannel(channel)
	defer c.clearChannel(channel)
	for {
		select {
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("video transcode delivery channel closed")
			}
			c.handleDelivery(delivery)
		case closeErr := <-closeCh:
			if closeErr != nil {
				return fmt.Errorf("video transcode channel closed: %w", closeErr)
			}
			return errors.New("video transcode channel closed")
		case <-c.stopCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *VideoTranscodeConsumer) handleDelivery(delivery amqp.Delivery) {
	var message VideoTranscodeMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil || strings.TrimSpace(message.MessageID) == "" {
		log.Printf("discard invalid video transcode message: %v", err)
		_ = delivery.Ack(false)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), videoTranscodeHandleTimeout)
	defer cancel()
	if err := c.handler.HandleVideoTranscodeMessage(ctx, message); err != nil {
		log.Printf("handle video transcode message failed: messageID=%s fileID=%s err=%v", message.MessageID, message.FileID, err)
		_ = delivery.Nack(false, true)
		return
	}
	_ = delivery.Ack(false)
}

func (c *VideoTranscodeConsumer) Close() {
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

func (c *VideoTranscodeConsumer) setChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channel = channel
}

func (c *VideoTranscodeConsumer) clearChannel(channel *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channel == channel {
		c.channel = nil
	}
}

func normalizeVideoTranscodeQueueName(queueName string) string {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return defaultVideoTranscodeQueue
	}
	return queueName
}
