package events

import (
	"telegram-service/pkg/client/mq"
	"telegram-service/pkg/logging"
)

type ProcessStrategy interface {
	Process(body []byte) error
}

type Worker struct {
	logger   *logging.Logger
	consumer mq.Consumer
	strategy ProcessStrategy
	messages <-chan mq.Message
}

func (w *Worker) Process() {
	for message := range w.messages {
		if err := w.strategy.Process(message.Body); err != nil {
			w.logger.Errorf("failed to Process a message %d due to error: %v", message.ID, err)
			if err != nil {
				w.reject(message.ID)
			}
			continue
		}

		w.ack(message.ID)
	}
}

func (w *Worker) ack(tag uint64) {
	if err := w.consumer.Ack(tag, false); err != nil {
		w.logger.Errorf("failed to Ack a message %d due to error: %v", tag, err)
	}
}

func (w *Worker) reject(tag uint64) {
	if err := w.consumer.Reject(tag, false); err != nil {
		w.logger.Errorf("failed to Reject a message %d due to error: %v", tag, err)
	}
}

func NewWorker(logger *logging.Logger, consumer mq.Consumer, strategy ProcessStrategy, messages <-chan mq.Message) *Worker {
	return &Worker{logger: logger, consumer: consumer, strategy: strategy, messages: messages}
}
