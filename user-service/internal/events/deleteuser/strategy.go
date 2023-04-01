package deleteuser

import (
	"context"
	"encoding/json"
	"fmt"
	"user-service/internal/events/model"
)

type ProcessStrategy struct {
	service model.Service
}

func (s *ProcessStrategy) Process(body []byte) ([]model.SendMessageRequest, error) {
	var request DeleteUserRequest

	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}
	if err := s.service.DeleteUser(context.Background(), request.UserID); err != nil {
		return nil, fmt.Errorf("service: %v", err)
	}

	response := model.SendMessageRequest{
		RequestID: request.RequestID,
		Message:   fmt.Sprintf("Удаление пользователя %d выполнено успешно, если он существовал.", request.UserID),
	}

	if request.SendResponse {
		return []model.SendMessageRequest{response}, nil
	}

	return nil, nil
}

func NewProcessStrategy(service model.Service) *ProcessStrategy {
	return &ProcessStrategy{service: service}
}
