package main

import (
	"context"
	"log"
	"ride-sharing/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq *messaging.Rabbitmq
}

func NewTripConsumer(rabbitmq *messaging.Rabbitmq) *tripConsumer {
	return &tripConsumer{
		rabbitmq: rabbitmq,
	}
}

func (c *tripConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages("hello", func(ctx context.Context, msg amqp.Delivery) error {
		log.Printf("driver received message: %v", msg)
		return nil
	})
}
