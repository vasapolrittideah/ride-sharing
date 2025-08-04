package messaging

import (
	"context"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
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
	msgs, err := r.Channel.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
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
				log.Fatalf("failed to handle the message: %v", err)
			}
		}
	}()

	return nil
}

func (r *Rabbitmq) PublishMessage(ctx context.Context, routingKey string, message string) error {
	return r.Channel.PublishWithContext(
		ctx,
		"",
		"hello",
		false,
		false,
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         []byte(message),
			DeliveryMode: amqp.Persistent,
		},
	)
}

func (r *Rabbitmq) setupExchangesAndQueues() error {
	_, err := r.Channel.QueueDeclare(
		"hello", // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		log.Fatal(err)
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
