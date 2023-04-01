package service

import (
	"encoding/json"
	"errors"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"telegram-service/internal/config"
	"telegram-service/internal/events/model"
	"telegram-service/pkg/client/mq"
	"telegram-service/pkg/logging"
)

const defaultRequestExpiration = "5000"

type Service struct {
	logger   *logging.Logger
	cfg      *config.Config
	bot      *tele.Bot
	producer mq.Producer
}

func NewService(logger *logging.Logger, cfg *config.Config, bot *tele.Bot, producer mq.Producer) *Service {
	return &Service{logger: logger, cfg: cfg, bot: bot, producer: producer}
}

func (s *Service) SendMessageWithOpts(id int64, msg string, opts ...interface{}) error {
	s.logger.Tracef("sending: id: %d, message: %s, opts: %d", id, msg, len(opts))

	chat := tele.ChatID(id)

	_, err := s.bot.Send(chat, msg, opts...)

	return s.middlewareError(id, err)
}

func (s *Service) EditMessageWithOpts(id int64, messageid int, msg string, opts ...interface{}) error {
	s.logger.Tracef("editing: userid: %d, messageid: %d, opts: %d", id, messageid, len(opts))

	_, err := s.bot.Edit(&EditableMessage{messageID: messageid, chatID: id}, msg, opts...)

	if errors.Is(err, tele.ErrTrueResult) {
		s.logger.Errorf("failed to edit a message due to error: %v", err)
		err = s.SendMessageWithOpts(id, s.cfg.Responses.BotError)
	}

	return s.middlewareError(id, err)
}

// MiddlewareError id - target user of this action
func (s *Service) middlewareError(id int64, err error) error {
	if err == nil {
		return nil
	}

	s.logger.Errorf("failed to execute bot instructions to user %d: %v", id, err)

	if errors.As(err, &tele.ErrBlockedByUser) || errors.As(err, &tele.ErrUserIsDeactivated) ||
		errors.As(err, &tele.ErrNotStartedByUser) {
		request := model.DeleteUserRequest{
			UserID: id,
		}
		requestBytes, err2 := json.Marshal(request)
		if err2 != nil {
			s.logger.Errorf("failed to json marshal a delete user request due to error: %v", err2)
			return err
		}
		if err2 = s.producer.Publish(s.cfg.RabbitMQ.Producer.UserExchange,
			s.cfg.RabbitMQ.Producer.DeleteUserRequestsKey, defaultRequestExpiration, requestBytes); err2 != nil {
			s.logger.Errorf("failed to publish a logout user request due to error: %v", err2)
			return err
		}
	}

	return err
}

type EditableMessage struct {
	messageID int
	chatID    int64
}

func (e *EditableMessage) MessageSig() (string, int64) {
	return fmt.Sprint(e.messageID), e.chatID
}
