package mq

type Queue interface {
	DeclareQueue(name string, durable, autoDelete, exclusive bool) error
}

type Exchange interface {
	DeclareExchange(name, kind string, durable, autoDelete, internal bool) error
	QueueBind(name, key, exchange string) error
	DeclareAndBindQueue(exchange, queue, key string) error
}

type Producer interface {
	Queue
	Exchange
	Publish(exchange, key, expiration string, msg []byte) error
}

type Consumer interface {
	Queue
	Consume(queue string, autoAck, exclusive bool) (<-chan Message, error)
	Ack(tag uint64, multiple bool) error
	Nack(tag uint64, multiple, requeue bool) error
	Reject(tag uint64, requeue bool) error
}

type Message struct {
	ID   uint64
	Body []byte
}
