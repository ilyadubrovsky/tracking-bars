package rabbitmq

import (
	"fmt"
	"github.com/streadway/amqp"
)

type Producer struct {
	*Session
}

func (p *Producer) Publish(exchange, key, expiration string, msg []byte) error {
	if !p.isReady {
		return errNotReady
	}

	if err := p.channel.Publish(
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         msg,
			Expiration:   expiration,
		}); err != nil {
		return err
	}

	return nil
}

func NewProducer(addr string) (*Producer, error) {
	session, err := New(addr)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq session: %v", err)
	}

	return &Producer{
		Session: session,
	}, nil
}
