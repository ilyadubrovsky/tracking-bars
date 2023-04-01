package news

import (
	"context"
	"encoding/json"
	"fmt"
	"user-service/internal/entity/user"
	"user-service/internal/events/model"
)

type ProcessStrategy struct {
	service  model.Service
	botError string
}

func (s *ProcessStrategy) Process(body []byte) ([]model.SendMessageRequest, error) {
	var request SendNewsRequest

	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	var (
		usrs []user.User
		err  error
	)

	switch request.Type {
	case "all":
		usrs, err = s.service.GetUsersIDByOpts(context.Background())
	case "auth":
		usrs, err = s.service.GetUsersIDByOpts(context.Background(), "WHERE deleted = false;")
	default:
		usrs, err = s.service.GetUsersIDByOpts(context.Background())
	}
	if err != nil {
		return nil, fmt.Errorf("service: %v", err)
	}

	responses := make([]model.SendMessageRequest, len(usrs))
	for i := range responses {
		responses[i] = model.SendMessageRequest{
			RequestID: usrs[i].ID,
			Message:   request.Message,
			ParseMode: request.ParseMode,
		}
	}

	return responses, nil
}

func NewProcessStrategy(service model.Service, botError string) *ProcessStrategy {
	return &ProcessStrategy{service: service, botError: botError}
}
