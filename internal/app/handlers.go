package app

import (
	"context"
	"errors"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"regexp"
	"strconv"
	"strings"
	"time"
	"tracking-barsv1.1/internal/entity/user"
)

var ErrIncorrectUserData = errors.New("the received user data is incorrect")

func (a *app) handleCallback(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callbackData := strings.Replace(c.Callback().Data, "\f", "", -1)
	if callbackData[:2] != "pt" {
		return c.Send(a.cfg.Responses.BotError)
	}

	usr, err := a.telegramService.GetUserByID(ctx, c.Sender().ID)
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	if usr == nil {
		return c.Edit(a.cfg.Responses.Bars.NotAuthorized)
	}

	if len(usr.ProgressTable.Tables) == 0 {
		return c.Edit(a.cfg.Responses.Bars.UnavailablePT)
	}

	if callbackData[2:] == "back" {
		progressTableInlineMarkup := makePtInlineMarkup(&usr.ProgressTable)

		msg := retrieveTablesData(usr.ProgressTable.Tables)
		return c.Edit(msg, "Markdown", progressTableInlineMarkup)
	}

	if callbackData[2:] == "show" {
		return c.Respond(&tele.CallbackResponse{
			Text: "Эта функция совсем скоро появится!",
		})
	}

	n, err := strconv.Atoi(callbackData[2:])
	if err != nil {
		return c.Edit(a.cfg.Responses.BotError)
	}

	if n <= 0 || n > len(usr.ProgressTable.Tables) {
		return c.Edit(a.cfg.Responses.Bars.UnavailablePT)
	}

	ptBackInlineMarkup := makePtBackInlineMarkup()

	return c.Edit(hideStNames(&usr.ProgressTable.Tables[n-1]).String(), "Markdown", ptBackInlineMarkup)
}

func (a *app) handleStartCommand(c tele.Context) error {
	return c.Send("Привет! Бот позволяет взаимодействовать с БАРС в телеграм. Вы можете смотреть оценки в удобной форме и получать уведомления об их изменениях. " +
		"Информация – /help.\n\nБот не является официальной разработкой НИУ «МЭИ».")
}

func (a *app) handleHelpCommand(c tele.Context) error {
	return c.Send("/auth Логин Пароль – авторизация в БАРС;\n" +
		"/pt – просмотр оценок в удобной форме;\n" +
		"/logout – удалить свои данные;\n" +
		"/gh – github репозиторий.")
}

func (a *app) handleAuthCommand(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if c.Message().Payload == "" {
		return c.Send(a.cfg.Responses.Bars.NoDataEntered)
	}

	if len(strings.Split(c.Message().Payload, " ")) != 2 {
		return c.Send(a.cfg.Responses.Bars.EntryFormIgnored)
	}

	userData := strings.Split(c.Message().Payload, " ")
	username := userData[0]
	password := userData[1]

	if err := validateUserData(username); errors.Is(err, ErrIncorrectUserData) {
		return c.Send(a.cfg.Responses.Bars.DataEnteredIncorrectly)
	}

	usr, err := a.telegramService.GetUserByID(ctx, c.Sender().ID)
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	if usr != nil && usr.ID == c.Sender().ID && usr.Deleted == false {
		return c.Send(a.cfg.Responses.Bars.AlreadyAuthorized)
	}

	status, err := a.barsService.Authorization(ctx, c.Sender().ID, username, password)
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	if !status {
		return c.Send(a.cfg.Responses.Bars.WrongData)
	}

	return c.Send(a.cfg.Responses.Bars.SuccessfulAuthorization)
}

func (a *app) handleLogoutCommand(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usr, err := a.telegramService.GetUserByID(ctx, c.Sender().ID)
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	if usr == nil {
		return c.Send(a.cfg.Responses.Bars.NotAuthorized)
	}

	if err = a.telegramService.LogoutUserByID(ctx, c.Sender().ID); err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	return c.Send(a.cfg.Responses.Bars.SuccessfulLogout)
}

func (a *app) handlePtCommand(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	progressTable, err := a.telegramService.GetProgressTableByID(ctx, c.Sender().ID)

	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	if progressTable == nil {
		return c.Send(a.cfg.Responses.Bars.NotAuthorized)
	}

	if len(progressTable.Tables) == 0 {
		return c.Send(a.cfg.Responses.Bars.UnavailablePT)
	}

	return c.Send(retrieveTablesData(progressTable.Tables),
		"Markdown",
		makePtInlineMarkup(progressTable),
	)
}

func (a *app) handleGhCommand(c tele.Context) error {
	return c.Send("Github репозиторий бота: [ссылка](github.com/ilyadubrovsky/tracking-bars).", "Markdown")
}

func (a *app) handleText(c tele.Context) error {
	return c.Send(a.cfg.Responses.Bars.DefaultAnswer)
}

func (a *app) handleEchoCommand(c tele.Context) error {
	msg := strings.Replace(c.Message().Text, "/echo ", "", -1)
	return c.Send(msg, "Markdown")
}

func (a *app) handleSendnewsAllCommand(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := strings.Replace(c.Message().Text, "/sendnewsall ", "", -1)

	users, err := a.telegramService.GetAllUsers(ctx)
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	for _, usr := range users {
		_, err = a.bot.Send(tele.ChatID(usr.ID), msg)
		if err != nil {
			a.logger.Errorf("failed to send a news due error: %s", err)
			a.logger.Debugf("UserID: %d, News: %s", usr.ID, msg)
		}
	}

	return nil
}

func (a *app) handleSendNewsAuthCommand(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := strings.Replace(c.Message().Text, "/sendnewsauth ", "", -1)

	users, err := a.telegramService.GetAllUsers(ctx, "WHERE deleted = false")
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	for _, usr := range users {
		_, err = a.bot.Send(tele.ChatID(usr.ID), msg)
		if err != nil {
			a.logger.Errorf("failed to send a news due error: %s", err)
			a.logger.Debugf("UserID: %d, News: %s", usr.ID, msg)
		}
	}

	return nil
}

func (a *app) handleSendmsgCommand(c tele.Context) error {
	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 3 {
		return c.Send("Команда не содержит отправлемое сообщение.")
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	text := strings.Join(msg[2:], " ")
	_, err = a.bot.Send(tele.ChatID(userID), text)
	if err != nil {
		return err
	}

	return c.Send(fmt.Sprintf("Пользователю %d успешно отправлено сообщение:\n %s", userID, text))
}

func (a *app) handleLogoutUserCommand(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 2 {
		return c.Send("Команда не содержит ID пользователя.")
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	if err = a.telegramService.LogoutUserByID(ctx, int64(userID)); err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	return c.Send(fmt.Sprintf("Удаление авторизационных данных пользователя %d прошло успешно.", userID))
}

func (a *app) handleDeleteUserCommand(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 2 {
		return c.Send("Команда не содержит ID пользователя.")
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	if err = a.telegramService.DeleteUserByID(ctx, int64(userID)); err != nil {
		return c.Send(a.cfg.Responses.BotError)
	}

	return c.Send(fmt.Sprintf("Удаление пользователя %d прошло успешно.", userID))
}

func retrieveTablesData(tables []user.SubjectTable) string {
	result := ""
	for i, subjectTable := range tables {
		result += fmt.Sprintf("*%d:* %s\n\n", i+1, subjectTable.Name)
	}
	result += "Для просмотра оценок по определённому предмету, пользуйтесь кнопочным меню."
	return result
}

func validateUserData(username string) error {
	var isStringAlphabeticAndBackslash = regexp.MustCompile(`^[a-zA-Z\\]+$`).MatchString
	if !isStringAlphabeticAndBackslash(username) {
		return ErrIncorrectUserData
	}
	return nil
}

func makePtInlineMarkup(ptObject *user.ProgressTable) *tele.ReplyMarkup {

	numberOfRows, numberOfButtonsInLastRow, numberOfButtonsInRow := 0, 0, 5
	if numberOfSubjects := len(ptObject.Tables); numberOfSubjects >= numberOfButtonsInRow {
		if remainder := numberOfSubjects % numberOfButtonsInRow; remainder == 0 {
			numberOfRows = numberOfSubjects / numberOfButtonsInRow
			numberOfButtonsInLastRow = numberOfButtonsInRow
		} else {
			numberOfRows = numberOfSubjects/numberOfButtonsInRow + 1
			numberOfButtonsInLastRow = remainder
		}
	} else if numberOfSubjects > 0 && numberOfSubjects < numberOfButtonsInRow {
		numberOfRows = 1
		numberOfButtonsInLastRow = numberOfSubjects
	} else {
		return &tele.ReplyMarkup{InlineKeyboard: make([][]tele.InlineButton, 0)}
	}

	keyboard := make([][]tele.InlineButton, numberOfRows)
	for i := range keyboard[:len(keyboard)-1] {
		keyboard[i] = make([]tele.InlineButton, numberOfButtonsInRow)
		for j := range keyboard[i] {
			data := fmt.Sprint(i*numberOfButtonsInRow + j + 1)
			keyboard[i][j] = tele.InlineButton{
				Unique: fmt.Sprint("pt", data),
				Text:   data,
			}
		}
	}

	keyboard[len(keyboard)-1] = make([]tele.InlineButton, numberOfButtonsInLastRow)
	for j := range keyboard[len(keyboard)-1] {
		data := fmt.Sprint((len(keyboard)-1)*numberOfButtonsInRow + j + 1)
		keyboard[len(keyboard)-1][j] = tele.InlineButton{
			Unique: fmt.Sprint("pt", data),
			Text:   data,
		}
	}

	return &tele.ReplyMarkup{
		InlineKeyboard: keyboard,
	}
}

func makePtBackInlineMarkup() *tele.ReplyMarkup {
	showButton := tele.InlineButton{
		Unique: "ptshow",
		Text:   "Показать названия КМов",
	}

	backButton := tele.InlineButton{
		Unique: "ptback",
		Text:   "Назад",
	}
	keyboard := make([][]tele.InlineButton, 2)
	keyboard[0] = append(keyboard[0], showButton)
	keyboard[1] = append(keyboard[1], backButton)
	return &tele.ReplyMarkup{
		InlineKeyboard: keyboard,
	}
}

func hideStNames(st *user.SubjectTable) *user.SubjectTable {
	for i := range st.Rows {
		if !(strings.HasPrefix(st.Rows[i].Name, "Балл текущего контроля") ||
			strings.HasPrefix(st.Rows[i].Name, "Итоговая оценка:") ||
			strings.HasPrefix(st.Rows[i].Name, "Промежуточная аттестация")) {
			st.Rows[i].Name = fmt.Sprintf("КМ-%d", i+1)
		}
	}
	return st
}
