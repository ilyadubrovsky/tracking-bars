package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"user-service/internal/config"
	"user-service/internal/entity/user"
	"user-service/internal/events/model"
	"user-service/internal/service"
)

type ProcessStrategy struct {
	service model.Service
	cfg     *config.Config
}

func (s *ProcessStrategy) Process(body []byte) ([]model.SendMessageRequest, error) {
	var request AuthorizationRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	dto := user.CreateUserDTO{
		ID:       request.UserID,
		Username: request.Username,
		Password: request.Password,
	}

	response := model.SendMessageRequest{
		RequestID: request.RequestID,
	}

	status, err := s.service.Authorization(context.Background(), dto)
	if errors.Is(err, service.ErrAlreadyAuthorized) {
		response.Message = s.cfg.Responses.Bars.AlreadyAuthorized
	} else if err != nil {
		response.Message = s.cfg.Responses.Bars.Error
	} else if !status {
		response.Message = s.cfg.Responses.Bars.WrongData
	} else {
		response.Message = s.cfg.Responses.Bars.SuccessfulAuthorization
	}

	return []model.SendMessageRequest{response}, nil

}

func NewProcessStrategy(service model.Service, cfg *config.Config) *ProcessStrategy {
	return &ProcessStrategy{service: service, cfg: cfg}
}
