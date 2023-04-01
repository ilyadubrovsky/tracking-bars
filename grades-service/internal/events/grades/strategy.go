package grades

import (
	"context"
	"encoding/json"
	"fmt"
	"grades-service/internal/events/model"
)

type ProcessStrategy struct {
	service       model.Service
	unavailablePT string
	notAuthorized string
	botError      string
}

func (s *ProcessStrategy) Process(body []byte) (*model.GetGradesResponse, error) {
	var request GetGradesRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	response := &model.GetGradesResponse{
		RequestID:    request.RequestID,
		IsCallback:   request.IsCallback,
		CallbackData: request.CallbackData,
		MessageID:    request.MessageID,
	}

	progressTable, err := s.service.GetProgressTableFromDB(context.Background(), request.RequestID)
	if err != nil {
		response.ResponseMessage = s.botError
		return response, fmt.Errorf("service: %v", err)
	}

	if progressTable == nil {
		response.ResponseMessage = s.notAuthorized
	} else if len(progressTable.Tables) == 0 {
		pt, err := s.service.UpdateAndGetProgressTable(context.Background(), request.RequestID)
		if err != nil {
			response.ResponseMessage = s.unavailablePT
			return response, fmt.Errorf("service: %v", err)
		} else {
			response.ProgressTable = *pt
		}
	} else {
		response.ProgressTable = *progressTable
	}

	return response, nil
}

func NewProcessStrategy(service model.Service, unavailablePT, notAuthorized, botError string) *ProcessStrategy {
	return &ProcessStrategy{service: service, unavailablePT: unavailablePT,
		notAuthorized: notAuthorized, botError: botError}
}
