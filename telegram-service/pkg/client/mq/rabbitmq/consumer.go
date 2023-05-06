package rabbitmq

import (
	"github.com/streadway/amqp"
	"telegram-service/pkg/client/mq"
)

type Consumer struct {
	*Session
	PrefetchCount int
}

func (c *Consumer) Consume(queue string, autoAck, exclusive bool) (<-chan mq.Message, error) {
	if !c.isReady {
		return nil, errNotReady
	}

	deliveries, err := c.consume(queue, autoAck, exclusive)
	if err != nil {
		return nil, err
	}

	message := make(chan mq.Message)

	go func() {
		for delivery := range deliveries {
			message <- mq.Message{
				ID:   delivery.DeliveryTag,
				Body: delivery.Body,
			}
			// TODO reconnect and close
		}
	}()

	return message, nil
}

func (c *Consumer) consume(queue string, autoAck, exclusive bool) (<-chan amqp.Delivery, error) {
	if err := c.channel.Qos(c.PrefetchCount, 0, false); err != nil {
		return nil, err
	}

	ch, err := c.channel.Consume(
		queue,
		"",
		autoAck,
		exclusive,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return ch, nil
}

func (c *Consumer) Ack(tag uint64, multiple bool) error {
	if err := c.channel.Ack(tag, multiple); err != nil {
		return err

	}

	return nil
}

func (c *Consumer) Nack(tag uint64, multiple, requeue bool) error {
	if err := c.channel.Nack(tag, multiple, requeue); err != nil {
		return err
	}

	return nil
}

func (c *Consumer) Reject(tag uint64, requeue bool) error {
	if err := c.channel.Reject(tag, requeue); err != nil {
		return err
	}

	return nil
}

func NewConsumer(addr string, prefetchCount int) (*Consumer, error) {
	session, err := New(addr)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		Session:       session,
		PrefetchCount: prefetchCount,
	}, nil
}
