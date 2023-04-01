package rabbitmq

import (
	"errors"
	"fmt"
	"github.com/streadway/amqp"
)

var (
	errNotReady     = errors.New("not ready")
	errNotConnected = errors.New("rabbitmq not connected")
)

type Session struct {
	connection *amqp.Connection
	channel    *amqp.Channel

	isConnected bool
	isReady     bool
}

func New(addr string) (*Session, error) {
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("connect: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open a channel: %v", err)
	}

	return &Session{
		connection:  conn,
		channel:     ch,
		isReady:     false,
		isConnected: true,
	}, nil
}

func (s *Session) QueueBind(name, key, exchange string) error {
	if !s.isConnected {
		return errNotConnected
	}

	if !s.isReady {
		return errNotReady
	}

	if err := s.channel.QueueBind(name, key, exchange, false, nil); err != nil {
		return err
	}

	return nil
}

func (s *Session) DeclareExchange(name, kind string, durable, autoDelete, internal bool) error {
	if !s.isConnected {
		return errNotConnected
	}

	if err := s.channel.ExchangeDeclare(name, kind, durable, autoDelete, internal, false, nil); err != nil {
		return err
	}

	return nil
}

func (s *Session) DeclareAndBindQueue(exchange, queue, key string) error {
	if err := s.DeclareQueue(queue, true, false, false); err != nil {
		return fmt.Errorf("queue: %v", err)
	}

	if err := s.QueueBind(queue, key, exchange); err != nil {
		return fmt.Errorf("bind a queue: %v", err)
	}

	return nil
}

func (s *Session) DeclareQueue(name string, durable, autoDelete, exclusive bool) error {
	if !s.isConnected {
		return errNotConnected
	}

	_, err := s.channel.QueueDeclare(name, durable, autoDelete, exclusive, false, nil)
	if err != nil {
		return err
	}

	s.isReady = true

	return nil
}
