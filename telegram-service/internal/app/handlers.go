package app

import (
	"encoding/json"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"os"
	"regexp"
	"strconv"
	"strings"
	"telegram-service/internal/events/model"
)

const defaultRequestExpiration = "5000"

func (a *App) handleCallback(c tele.Context) error {
	callbackData := strings.Replace(c.Callback().Data, "\f", "", -1)
	if callbackData[:2] != "pt" {
		return a.service.EditMessageWithOpts(c.Sender().ID, c.Message().ID, a.cfg.Responses.BotError)
	}

	request := model.GetGradesRequest{
		RequestID:    c.Sender().ID,
		IsCallback:   true,
		CallbackData: callbackData[2:],
		MessageID:    c.Message().ID,
	}

	if err := a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.GradesExchange, a.cfg.RabbitMQ.Producer.GradesRequestsKey); err != nil {
		a.logger.Errorf("failed to handle a callback due to error: %v", err)
		return a.service.EditMessageWithOpts(c.Sender().ID, c.Message().ID, a.cfg.Responses.BotError)
	}

	return nil
}

func (a *App) handleStartCommand(c tele.Context) error {
	return a.service.SendMessageWithOpts(c.Sender().ID,
		"Привет! Бот позволяет взаимодействовать с БАРС в телеграм. Вы можете смотреть оценки в удобной форме "+
			"и получать уведомления об их изменениях. Информация – /help.\n\nБот не является официальной разработкой НИУ «МЭИ».")
}

func (a *App) handleHelpCommand(c tele.Context) error {
	return a.service.SendMessageWithOpts(c.Sender().ID, "/auth Логин Пароль – авторизация в БАРС;\n"+
		"/pt – просмотр оценок в удобной форме;\n"+
		"/logout – удалить свои данные;\n"+
		"/sq Текст – отправить любое обращение (например, об ошибке при некорректном отображении данных);\n"+
		"/gh – github репозиторий.")
}

func (a *App) handleAuthCommand(c tele.Context) error {
	if c.Message().Payload == "" {
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.Bars.NoDataEntered)
	}

	userData := strings.Split(c.Message().Payload, " ")

	if len(userData) != 2 {
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.Bars.EntryFormIgnored)
	}

	username := userData[0]
	password := userData[1]

	if !validateUserData(username) {
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.Bars.DataEnteredIncorrectly)
	}

	request := model.AuthorizationRequest{
		RequestID: c.Sender().ID,
		UserID:    c.Sender().ID,
		Username:  username,
		Password:  password,
	}

	if err := a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.UserExchange, a.cfg.RabbitMQ.Producer.AuthRequestsKey); err != nil {
		a.logger.Errorf("failed to handle auth command due to error: %v", err)
		a.logger.Debugf("Request: RequestID: %d, UserID: %d, Username: %s", request.RequestID, request.UserID, request.Username)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	return nil
}

func (a *App) handleLogoutCommand(c tele.Context) error {
	request := model.LogoutRequest{
		RequestID: c.Sender().ID,
		UserID:    c.Sender().ID,
	}

	if err := a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.UserExchange, a.cfg.RabbitMQ.Producer.LogoutRequestsKey); err != nil {
		a.logger.Errorf("failed to handle logout command due to error: %v", err)
		a.logger.Debugf("Request: RequestID: %d, UserID: %d", request.RequestID, request.UserID)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	return nil
}

func (a *App) handlePtCommand(c tele.Context) error {
	request := model.GetGradesRequest{
		RequestID:  c.Sender().ID,
		IsCallback: false,
	}

	if err := a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.GradesExchange, a.cfg.RabbitMQ.Producer.GradesRequestsKey); err != nil {
		a.logger.Errorf("failed to handle pt command due to error: %v", err)
		a.logger.Debugf("Request: RequestID: %d, IsCallback: %v, CallbackData :%s, MessageID: %d",
			request.RequestID, request.IsCallback, request.CallbackData, request.MessageID)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	return nil
}

func (a *App) handleGhCommand(c tele.Context) error {
	return a.service.SendMessageWithOpts(c.Sender().ID,
		"Github репозиторий бота: [ссылка](github.com/ilyadubrovsky/tracking-bars).", tele.ModeMarkdown)
}

func (a *App) handleSqCommand(c tele.Context) error {
	msg := strings.Replace(c.Message().Text, "/sq", "", -1)

	if len(strings.Split(c.Message().Text, " ")) < 2 {
		return a.service.SendMessageWithOpts(c.Sender().ID, "Обращение не было указано, его необходимо написать через пробел после команды.")
	}

	if msg == "" {
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	adminID, err := strconv.Atoi(os.Getenv("ADMIN_ID"))
	if err != nil {
		a.logger.Errorf("failed to handle sq command due to error: %v", err)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	if err = a.service.SendMessageWithOpts(int64(adminID),
		fmt.Sprintf("Обращение от пользователя %d:\n%s", c.Sender().ID, msg)); err != nil {
		if err2 := a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError); err2 != nil {
			return err2
		}
		return err
	}

	return a.service.SendMessageWithOpts(c.Sender().ID, "Ваше обращение успешно отправлено.")
}

func (a *App) handleText(c tele.Context) error {
	return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.Bars.DefaultAnswer)
}

func (a *App) handleEchoCommand(c tele.Context) error {
	msg := strings.Replace(c.Message().Text, "/echo ", "", -1)
	return a.service.SendMessageWithOpts(c.Sender().ID, msg, tele.ModeMarkdown)
}

func (a *App) handleSendnewsAllCommand(c tele.Context) error {
	msg := strings.Replace(c.Message().Text, "/sendnewsall ", "", -1)

	request := model.SendNewsRequest{
		RequestID: c.Sender().ID,
		Type:      "all",
		Message:   msg,
		ParseMode: tele.ModeMarkdown,
	}

	if err := a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.UserExchange, a.cfg.RabbitMQ.Producer.NewsRequestsKey); err != nil {
		a.logger.Errorf("failed to handle sendnews command due to error: %v", err)
		a.logger.Debugf("Request: RequestID: %d, Type: %s, Message: %s ParseMode: %s",
			request.RequestID, request.Type, request.Message, request.ParseMode)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	return a.service.SendMessageWithOpts(c.Sender().ID, fmt.Sprintf("Сообщение:\n%s\nуспешно отправлено ВСЕМ пользователям.", msg), tele.ModeMarkdown)
}

func (a *App) handleSendNewsAuthCommand(c tele.Context) error {
	msg := strings.Replace(c.Message().Text, "/sendnewsauth ", "", -1)

	request := model.SendNewsRequest{
		RequestID: c.Sender().ID,
		Type:      "auth",
		Message:   msg,
		ParseMode: tele.ModeMarkdown,
	}

	if err := a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.UserExchange, a.cfg.RabbitMQ.Producer.NewsRequestsKey); err != nil {
		a.logger.Errorf("failed to handle sendnewsauth command due to error: %v", err)
		a.logger.Debugf("Request: RequestID: %d, Type: %s, Message: %s ParseMode: %s",
			request.RequestID, request.Type, request.Message, request.ParseMode)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	return a.service.SendMessageWithOpts(c.Sender().ID, fmt.Sprintf("Сообщение:\n%s\nуспешно отправлено АВТОРИЗОВАННЫМ пользователям.", msg), tele.ModeMarkdown)
}

func (a *App) handleLogoutUserCommand(c tele.Context) error {
	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 2 {
		return a.service.SendMessageWithOpts(c.Sender().ID, "Команда не содержит UserID пользователя.")
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	request := model.LogoutRequest{
		RequestID: c.Sender().ID,
		UserID:    int64(userID),
	}

	if err = a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.UserExchange, a.cfg.RabbitMQ.Producer.LogoutRequestsKey); err != nil {
		a.logger.Errorf("failed to handle logoutuser command due to error: %v", err)
		a.logger.Debugf("Request: RequestID: %d, UserID: %d", request.RequestID, request.UserID)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	return a.service.SendMessageWithOpts(c.Sender().ID, fmt.Sprintf("Запрос на деавторизацию пользователя %d принят и ответ будет отправлен в ближайшее время.", request.UserID))
}

func (a *App) handleDeleteUserCommand(c tele.Context) error {
	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 2 {
		return a.service.SendMessageWithOpts(c.Sender().ID, "Команда не содержит UserID пользователя.")
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	request := model.DeleteUserRequest{
		RequestID:    c.Sender().ID,
		UserID:       int64(userID),
		SendResponse: true,
	}

	if err = a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.UserExchange, a.cfg.RabbitMQ.Producer.DeleteUserRequestsKey); err != nil {
		a.logger.Errorf("failed to handle deleteuser command due to error: %v", err)
		a.logger.Debugf("Request: RequestID: %d, UserID: %d, SendResponse: %v", request.RequestID, request.UserID, request.SendResponse)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	return a.service.SendMessageWithOpts(c.Sender().ID, fmt.Sprintf("Запрос на удаление пользователя %d принят и ответ будет отправлен в ближайшее время.", request.UserID))
}

func (a *App) handleAuthUserCommand(c tele.Context) error {
	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 4 {
		return a.service.SendMessageWithOpts(c.Sender().ID, "Команда не содержит все необходимые данные.")
	}

	id, err := strconv.Atoi(msg[1])
	if err != nil {
		return a.service.SendMessageWithOpts(c.Sender().ID, "Получен некорректный ID.")
	}

	if !validateUserData(msg[2]) {
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.Bars.DataEnteredIncorrectly)
	}

	request := model.AuthorizationRequest{
		RequestID: c.Sender().ID,
		UserID:    int64(id),
		Username:  msg[2],
		Password:  msg[3],
	}

	if err = a.marshalAndPublish(request, a.cfg.RabbitMQ.Producer.UserExchange, a.cfg.RabbitMQ.Producer.AuthRequestsKey); err != nil {
		a.logger.Errorf("failed to handle deleteuser command due to error: %v", err)
		a.logger.Debugf("Request: RequestID: %d, UserID: %d, Username: %s", request.RequestID, request.UserID, request.Username)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	return a.service.SendMessageWithOpts(c.Sender().ID, fmt.Sprintf("Запрос на авторизацию пользователя %d принят и ответ будет отправлен в ближайшее время.", request.UserID))
}

func (a *App) handleFixGradesCommand(c tele.Context) error {
	return a.service.SendMessageWithOpts(c.Sender().ID,
		"Ваши оценки не могут быть получены, поскольку страница с оценками не является основной страницей в Вашем аккаунте БАРС."+
			"\n\n*Для того, чтобы это исправить и бот заработал, выполните следующие действия:*\n"+
			"*1.* Зайдите в БАРС (через браузер телефона, компьютера или иным способом);\n"+
			"*2.* Зайдите на страницу оценок (именно в раздел \"Оценки БАРС\", а не \"Сводка\";\n"+
			"*3.* Нажмите на значок шестерёнки в верхнем меню страницы (правый верхний угол), затем на кнопку \"Установить\";\n"+
			"*4.* Выполните авторизацию в боте повторно, всё должно заработать.\n\n"+
			"Если возникнут вопросы или эти действия не помогут, Вы можете написать своё обращение с помощью команды /sq.", tele.ModeMarkdown)
}

func (a *App) handleSendmsgCommand(c tele.Context) error {
	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 3 {
		return a.service.SendMessageWithOpts(c.Sender().ID, "Команда не содержит отправлемое сообщение или ID пользователя.")
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		a.logger.Errorf("failed to handle sendmsg command due to error: %v", err)
		return a.service.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)
	}

	text := strings.Join(msg[2:], " ")
	if err = a.service.SendMessageWithOpts(int64(userID), text, tele.ModeMarkdown); err != nil {
		return err
	}

	return a.service.SendMessageWithOpts(c.Sender().ID,
		fmt.Sprintf("Пользователю %d успешно отправлено сообщение:\n%s", userID, text), tele.ModeMarkdown)
}

func (a *App) marshalAndPublish(request interface{}, exchange, key string) error {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("json marshal: %v", err)
	}

	if err = a.producer.Publish(exchange, key, defaultRequestExpiration, requestBytes); err != nil {
		return fmt.Errorf("producer publish: %v", err)
	}

	return nil
}

func validateUserData(username string) bool {
	var isStringAlphabeticAndBackslash = regexp.MustCompile(`^[a-zA-Z\\]+$`).MatchString
	if !isStringAlphabeticAndBackslash(username) {
		return false
	}
	return true
}
