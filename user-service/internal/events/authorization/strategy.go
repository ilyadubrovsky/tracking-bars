package authorization

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ilyadubrovsky/bars"
	"user-service/internal/apperror"
	"user-service/internal/config"
	"user-service/internal/entity/user"
	"user-service/internal/events/model"
)

type userAuthorizer interface {
	Authorization(ctx context.Context, dto user.AuthorizationUserDTO) error
}

type ProcessStrategy struct {
	service userAuthorizer
	cfg     *config.Config
}

func (s *ProcessStrategy) Process(body []byte) ([]model.SendMessageRequest, error) {
	var request AuthorizationRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	dto := user.AuthorizationUserDTO{
		ID:       request.UserID,
		Username: request.Username,
		Password: request.Password,
	}

	response := model.SendMessageRequest{
		RequestID: request.RequestID,
	}

	if err := s.service.Authorization(context.Background(), dto); err != nil {
		switch err {
		case apperror.ErrAlreadyAuthorized:
			response.Message = s.cfg.Responses.Bars.AlreadyAuthorized
		case bars.ErrWrongGradesPage:
			response.Message = s.cfg.Responses.Bars.WrongGradesPage
		case bars.ErrNoAuth:
			response.Message = s.cfg.Responses.Bars.WrongData
		default:
			response.Message = s.cfg.Responses.Bars.Error
		}
	} else {
		response.Message = s.cfg.Responses.Bars.SuccessfulAuthorization
	}

	return []model.SendMessageRequest{response}, nil

}

func NewProcessStrategy(service userAuthorizer, cfg *config.Config) *ProcessStrategy {
	return &ProcessStrategy{service: service, cfg: cfg}
}
