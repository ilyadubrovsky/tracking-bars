package tgmessages

import (
	"encoding/json"
	"fmt"
)

type messageSender interface {
	SendMessageWithOpts(id int64, msg string, opts ...interface{}) error
}

type ProcessStrategy struct {
	service messageSender
}

func (s *ProcessStrategy) Process(body []byte) error {
	var request SendMessageRequest

	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("json unmarshal: %v", err)
	}

	return s.service.SendMessageWithOpts(request.RequestID, request.Message, request.ParseMode)
}

func NewProcessStrategy(service messageSender) *ProcessStrategy {
	return &ProcessStrategy{service: service}
}
