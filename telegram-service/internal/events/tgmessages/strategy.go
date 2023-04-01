package tgmessages

import (
	"encoding/json"
	"fmt"
	"telegram-service/internal/events/model"
)

type ProcessStrategy struct {
	service model.Service
}

func (s *ProcessStrategy) Process(body []byte) error {
	var request SendMessageRequest

	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("json unmarshal: %v", err)
	}

	return s.service.SendMessageWithOpts(request.RequestID, request.Message, request.ParseMode)
}

func NewProcessStrategy(service model.Service) *ProcessStrategy {
	return &ProcessStrategy{service: service}
}
