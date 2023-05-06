package logout

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"user-service/internal/config"
	"user-service/internal/events/model"
	"user-service/internal/service"
)

type userLogouter interface {
	Logout(ctx context.Context, id int64) error
}

type ProcessStrategy struct {
	service userLogouter
	cfg     *config.Config
}

func (s *ProcessStrategy) Process(body []byte) ([]model.SendMessageRequest, error) {

	var request LogoutRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	response := model.SendMessageRequest{
		RequestID: request.RequestID,
	}

	err := s.service.Logout(context.Background(), request.UserID)
	if errors.Is(err, service.ErrNotAuthorized) {
		response.Message = s.cfg.Responses.Bars.NotAuthorized
	} else if err != nil {
		response.Message = s.cfg.Responses.BotError
	} else {
		response.Message = s.cfg.Responses.Bars.SuccessfulLogout
	}

	return []model.SendMessageRequest{response}, nil
}

func NewProcessStrategy(service userLogouter, cfg *config.Config) *ProcessStrategy {
	return &ProcessStrategy{service: service, cfg: cfg}
}
