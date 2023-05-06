package events

import (
	"encoding/json"
	"fmt"
	"grades-service/internal/events/model"
	"grades-service/pkg/client/mq"
	"grades-service/pkg/logging"
)

const defaultRequestExpiration = "5000"

type ProcessStrategy interface {
	Process(body []byte) (*model.GetGradesResponse, error)
}

type Worker struct {
	logger            *logging.Logger
	consumer          mq.Consumer
	producer          mq.Producer
	responsesExchange string
	responsesKey      string
	strategy          ProcessStrategy
	messages          <-chan mq.Message
}

func (w *Worker) Process() {
	for message := range w.messages {
		response, err := w.strategy.Process(message.Body)
		if err != nil {
			w.logger.Errorf("failed to Process a message %d due to error: %v", message.ID, err)
			if response == nil {
				w.reject(message.ID)
				continue
			}
		}

		if err = w.sendResponse(response); err != nil {
			w.logger.Errorf("failed to send a response due to error: %v", err)
			w.logger.Tracef("rabbitmq messageID: %d", message.ID)
			w.logger.Debugf("Response: RequestID: %d ProgressTable: %s, ResponseMessage: %s IsCallback: %v CallbackData: %s, MessageID: %d",
				response.RequestID, response.ProgressTable, response.ResponseMessage, response.IsCallback, response.CallbackData, response.MessageID)
			w.reject(message.ID)
			continue
		}

		w.ack(message.ID)
	}
}

func (w *Worker) sendResponse(response interface{}) error {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("json marshal: %v", err)
	}

	if err = w.producer.Publish(w.responsesExchange, w.responsesKey, defaultRequestExpiration, responseBytes); err != nil {
		return err
	}

	return nil
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

func NewWorker(logger *logging.Logger, consumer mq.Consumer, producer mq.Producer, responsesExchange string, responsesKey string, strategy ProcessStrategy, messages <-chan mq.Message) *Worker {
	return &Worker{logger: logger, consumer: consumer, producer: producer,
		responsesExchange: responsesExchange, responsesKey: responsesKey, strategy: strategy, messages: messages}
}
