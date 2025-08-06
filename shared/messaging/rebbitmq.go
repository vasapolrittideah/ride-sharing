package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/shared/contracts"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TripExchange = "trip"
)

type Rabbitmq struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewRebbitmq(uri string) (*Rabbitmq, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %v", err)
	}

	rmq := &Rabbitmq{
		conn:    conn,
		Channel: ch,
	}

	if err := rmq.setupExchangesAndQueues(); err != nil {
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}

	return rmq, nil
}

type MessageHandler func(context.Context, amqp.Delivery) error

func (r *Rabbitmq) ConsumeMessages(queueName string, handler MessageHandler) error {
	// Set prefetch count to 1 for fair dispatch
	// This tells RabbitMQ not to give more than one message to a service at a time.
	// The worker will only get the next message after it has acknowledged the previous one.
	err := r.Channel.Qos(
		1,     // prefetchCount: Limit to 1 unacknowledged message per consumer
		0,     // prefetchSize: No specific limit on message size
		false, // global: Apply prefetchCount to each consumer individually
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

	msgs, err := r.Channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	ctx := context.Background()

	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)

			if err := handler(ctx, msg); err != nil {
				log.Printf("ERROR: Failed to handle message: %v. Message body: %s", err, msg.Body)
				// Nack the message. Set requeue to false to avoid immediate redelivery loops.
				// Consider a dead-letter exchange (DLQ) or a more sophisticated retry mechanism for production.
				if nackErr := msg.Nack(false, false); nackErr != nil {
					log.Printf("ERROR: Failed to Nack message: %v", nackErr)
				}

				// Continue to the next message
				continue
			}

			if ackErr := msg.Ack(false); ackErr != nil {
				log.Printf("ERROR: Failed to Ack message: %v. Message body: %s", ackErr, msg.Body)
			}
		}
	}()

	return nil
}

func (r *Rabbitmq) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {
	log.Printf("Publishing message with routing key: %s", routingKey)

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	return r.Channel.PublishWithContext(
		ctx,
		TripExchange, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         jsonMessage,
			DeliveryMode: amqp.Persistent,
		},
	)
}

func (r *Rabbitmq) setupExchangesAndQueues() error {
	err := r.Channel.ExchangeDeclare(
		TripExchange, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %s: %v", TripExchange, err)
	}

	if err := r.declareAndBindQueue(
		FindAvailableDriversQueue,
		[]string{
			contracts.TripEventCreated,
			contracts.TripEventDriverNotInterested,
		},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		DriverCmdTripRequestQueue,
		[]string{
			contracts.DriverCmdTripRequest,
		},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		DriverTripResponseQueue,
		[]string{
			contracts.DriverCmdTripAccept,
			contracts.DriverCmdTripDecline,
		},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyDriverNoDriversFoundQueue,
		[]string{
			contracts.TripEventNoDriversFound,
		},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyDriverAssignedQueue,
		[]string{
			contracts.TripEventDriverAssigned,
		},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		PaymentTripResponseQueue,
		[]string{
			contracts.PaymentCmdCreateSession,
		},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyPaymentSessionCreatedQueue,
		[]string{
			contracts.PaymentEventSessionCreated,
		},
		TripExchange,
	); err != nil {
		return err
	}

	return nil
}

func (r *Rabbitmq) declareAndBindQueue(queueName string, messageTypes []string, exchange string) error {
	q, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, msg := range messageTypes {
		if err := r.Channel.QueueBind(
			q.Name,
			msg,
			exchange,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to %s: %v", queueName, err)
		}
	}

	return nil
}

func (r *Rabbitmq) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
	if r.Channel != nil {
		r.Channel.Close()
	}
}
