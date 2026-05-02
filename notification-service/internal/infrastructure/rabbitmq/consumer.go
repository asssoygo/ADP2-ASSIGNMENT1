package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	mainQueue  = "payment.completed"
	dlxName    = "payment.dlx"
	dlqQueue   = "payment.dead"
	maxRetries = 3
	msgTTLms   = int32(60000) // 1 minute safety TTL on main queue
)

type Consumer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	dlqChannel *amqp.Channel
	processor  usecase.EventProcessor
	retries    map[string]int
	mu         sync.Mutex
}

func NewConsumer(url string, processor usecase.EventProcessor) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// ── Dead Letter Exchange ──────────────────────────────────────────────────
	if err := ch.ExchangeDeclare(
		dlxName,  // name
		"direct", // type
		true,     // durable
		false,    // auto-delete
		false,    // internal
		false,    // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare DLX: %w", err)
	}

	// ── Dead Letter Queue ─────────────────────────────────────────────────────
	if _, err := ch.QueueDeclare(
		dlqQueue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare DLQ: %w", err)
	}

	if err := ch.QueueBind(dlqQueue, dlqQueue, dlxName, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("bind DLQ to DLX: %w", err)
	}

	// ── Main Queue with DLQ configuration ────────────────────────────────────
	// x-dead-letter-exchange routes rejected/expired messages to the DLX.
	// x-message-ttl is a safety net: messages not consumed within 1 min are dead-lettered.
	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    dlxName,
		"x-dead-letter-routing-key": dlqQueue,
		"x-message-ttl":             msgTTLms,
	}

	ch, err = declareMainQueue(conn, ch, queueArgs)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("declare main queue: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	// Separate channel so DLQ consumer doesn't share state with main channel.
	dlqCh, err := conn.Channel()
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &Consumer{
		conn:       conn,
		channel:    ch,
		dlqChannel: dlqCh,
		processor:  processor,
		retries:    make(map[string]int),
	}, nil
}

// declareMainQueue declares payment.completed with DLQ args.
// If the queue already exists with different args (406 PRECONDITION_FAILED),
// the broker closes the channel; we open a fresh one, delete the stale queue,
// and re-declare with the correct args.
func declareMainQueue(conn *amqp.Connection, ch *amqp.Channel, args amqp.Table) (*amqp.Channel, error) {
	_, err := ch.QueueDeclare(mainQueue, true, false, false, false, args)
	if err == nil {
		return ch, nil
	}

	// Channel was closed by broker after the 406 — open a replacement.
	_ = ch.Close()
	ch2, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("reopen channel after 406: %w", err)
	}

	// Delete the queue so it can be re-declared with the new args.
	if _, err := ch2.QueueDelete(mainQueue, false, false, false); err != nil {
		ch2.Close()
		return nil, fmt.Errorf("delete stale queue: %w", err)
	}

	if _, err := ch2.QueueDeclare(mainQueue, true, false, false, false, args); err != nil {
		ch2.Close()
		return nil, fmt.Errorf("redeclare with DLQ args: %w", err)
	}

	log.Printf("[Notification] Recreated %q queue with DLQ configuration", mainQueue)
	return ch2, nil
}

// Start blocks, consuming from the main queue and monitoring the DLQ.
// It returns when the connection is closed (graceful shutdown).
func (c *Consumer) Start() error {
	// ── DLQ monitor ──────────────────────────────────────────────────────────
	dlqMsgs, err := c.dlqChannel.Consume(dlqQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume DLQ: %w", err)
	}
	go func() {
		for msg := range dlqMsgs {
			var event domain.PaymentEvent
			if jsonErr := json.Unmarshal(msg.Body, &event); jsonErr != nil {
				log.Printf("[DLQ] Received unreadable message, discarding")
			} else {
				log.Printf("[DLQ] Message %s moved to dead letter queue after %d attempts",
					event.MessageID, maxRetries)
			}
			msg.Ack(false)
		}
	}()

	// ── Main queue consumer ───────────────────────────────────────────────────
	msgs, err := c.channel.Consume(mainQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume main queue: %w", err)
	}

	log.Printf("[Notification] Waiting for messages on queue: %s (DLQ: %s)", mainQueue, dlqQueue)

	for msg := range msgs {
		var event domain.PaymentEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("[Notification] Malformed message, discarding: %v", err)
			msg.Nack(false, false) // discard — cannot retry without a valid ID
			continue
		}

		if err := c.processor.ProcessPaymentEvent(event); err != nil {
			c.mu.Lock()
			c.retries[event.MessageID]++
			attempt := c.retries[event.MessageID]
			c.mu.Unlock()

			if attempt >= maxRetries {
				log.Printf("[Notification] Message %s failed %d/%d times — routing to DLQ",
					event.MessageID, attempt, maxRetries)
				c.mu.Lock()
				delete(c.retries, event.MessageID)
				c.mu.Unlock()
				msg.Nack(false, false) // dead-letter: no requeue
			} else {
				log.Printf("[Notification] Message %s attempt %d/%d failed, requeuing: %v",
					event.MessageID, attempt, maxRetries, err)
				msg.Nack(false, true) // requeue for next attempt
			}
			continue
		}

		// Success — clear retry counter and acknowledge
		c.mu.Lock()
		delete(c.retries, event.MessageID)
		c.mu.Unlock()
		msg.Ack(false)
	}

	return nil
}

func (c *Consumer) Close() {
	if c.dlqChannel != nil {
		c.dlqChannel.Close()
	}
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
