package events

import (
	"context"
	"ride-sharing/shared/messaging"
)

type TripEventPublisher struct {
	rabbitmq *messaging.Rabbitmq
}

func NewTripEventPublisher(rabbitmq *messaging.Rabbitmq) *TripEventPublisher {
	return &TripEventPublisher{
		rabbitmq: rabbitmq,
	}
}

func (p *TripEventPublisher) PublishTripCreated(ctx context.Context) error {
	return p.rabbitmq.PublishMessage(ctx, "hello", "Hello World!")
}
