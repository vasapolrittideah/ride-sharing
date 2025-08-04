package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Rabbitmq struct {
	conn *amqp.Connection
}

func NewRebbitmq(uri string) (*Rabbitmq, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	rmq := &Rabbitmq{
		conn: conn,
	}

	return rmq, nil
}

func (r *Rabbitmq) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}
