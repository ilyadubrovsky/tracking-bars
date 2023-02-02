package app

import (
	"TrackingBARSv2/internal/entity/user"
	"TrackingBARSv2/pkg/client/bars"
	"TrackingBARSv2/pkg/utils/aes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var ErrIncorrectUserData = errors.New("the received user data is incorrect")

func (a *app) handleCallback(c tele.Context) error {
	callbackData := strings.Replace(c.Callback().Data, "\f", "", -1)
	if callbackData[:2] != "pt" {
		return c.Send(a.cfg.Messages.BotError)
	}

	usr, err := a.usersStorage.FindOne(context.Background(), c.Sender().ID)
	if err != nil {
		a.logger.Errorf("failed to FindOne due error: %s", err)
		return c.Send(a.cfg.Messages.BotError)
	}

	if usr.Deleted == true || usr.ID != c.Sender().ID {
		return c.Edit(a.cfg.Messages.Bars.NotAuthorized)
	}

	if len(usr.ProgressTable.Tables) == 0 {
		return c.Edit(a.cfg.Messages.Bars.UnavailablePT)
	}

	if callbackData[2:] == "back" {
		progressTableInlineMarkup := makePtInlineMarkup(&usr.ProgressTable)

		msg := retrieveTablesData(usr.ProgressTable.Tables)
		return c.Edit(msg, "Markdown", progressTableInlineMarkup)
	}

	n, err := strconv.Atoi(callbackData[2:])
	if err != nil {
		return c.Edit(a.cfg.Messages.BotError)
	}

	if n <= 0 || n > len(usr.ProgressTable.Tables) {
		return c.Edit(a.cfg.Messages.Bars.UnavailablePT)
	}

	ptBackInlineMarkup := makePtBackInlineMarkup()
	return c.Edit(usr.ProgressTable.Tables[n-1].String(), "Markdown", ptBackInlineMarkup)
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
	if c.Message().Payload == "" {
		return c.Send(a.cfg.Messages.Bars.NoDataEntered)
	}

	if len(strings.Split(c.Message().Payload, " ")) != 2 {
		return c.Send(a.cfg.Messages.Bars.EntryFormIgnored)
	}

	userData := strings.Split(c.Message().Payload, " ")
	username := userData[0]
	password := userData[1]

	if err := validateUserData(username); err == ErrIncorrectUserData {
		return c.Send(a.cfg.Messages.Bars.DataEnteredIncorrectly)
	}

	usr, err := a.usersStorage.FindOne(context.Background(), c.Sender().ID)
	if err != nil {
		a.logger.Errorf("failed to FindOne due error: %s", err)
		return c.Send(a.cfg.Messages.BotError)
	}
	if usr.Deleted == false && usr.ID == c.Sender().ID {
		return c.Send(a.cfg.Messages.Bars.AlreadyAuthorized)
	}

	client := bars.NewClient(a.cfg.Bars.URLs.RegistrationURL)

	err = client.Authorization(context.Background(), username, password)

	if err == bars.ErrNoAuth {
		return c.Send(a.cfg.Messages.Bars.WrongData)
	}
	if err != nil {
		a.logger.Errorf("failed to Authorization the user due error: %s", err)
		return c.Send(a.cfg.Messages.Bars.Error)
	}

	encryptedPassword, err := aes.EncryptAES([]byte(os.Getenv("ENCRYPTION_KEY")), []byte(password))
	if err != nil {
		a.logger.Warningf("failed to encrypt (EncryptAES) a password due error: %s", err)
		a.logger.Debugf("UserID: %d\n", c.Sender().ID)
		return c.Send(a.cfg.Messages.BotError)
	}

	pt, err := a.barsService.GetProgressTable(context.Background(), client)
	if err != nil {
		a.logger.Errorf("failed to GetProgressTables due error: %s", err)
		a.logger.Debugf("UserID %d", c.Sender().ID)
		return c.Send(a.cfg.Messages.BotError)
	}

	ptBytes, err := json.Marshal(pt)
	if err != nil {
		a.logger.Errorf("failed to marshal a struct due error: %s", err)
		a.logger.Debugf("UserID: %d\n Progress Table: %s\n", usr.ID, pt)
		return c.Send(a.cfg.Messages.BotError)
	}

	if usr.Deleted == true {
		usrDTO := user.UpdateUserDTO{
			ID:            c.Sender().ID,
			Username:      username,
			Password:      encryptedPassword,
			ProgressTable: string(ptBytes),
			Deleted:       false,
		}
		err = a.usersStorage.Update(context.Background(), usrDTO)
	} else {
		usrDTO := user.CreateUserDTO{
			ID:            c.Sender().ID,
			Username:      username,
			Password:      encryptedPassword,
			ProgressTable: string(ptBytes),
		}
		err = a.usersStorage.Create(context.Background(), usrDTO)
	}
	if err != nil {
		a.logger.Errorf("failed to Create/Update due error: %s", err)
		a.logger.Debugf("User ID: %d, Username: %s, \n EncryptedPassword: %s, \n", usr.ID, usr.Username, usr.Password)
		return c.Send(a.cfg.Messages.BotError)
	}

	return c.Send(a.cfg.Messages.Bars.SuccessfulAuthorization)
}

func (a *app) handleLogoutCommand(c tele.Context) error {
	usr, err := a.usersStorage.FindOne(context.Background(), c.Sender().ID)
	if err != nil {
		a.logger.Errorf("failed to FindOne due error: %s", err)
		return c.Send(a.cfg.Messages.BotError)
	}

	if usr.Deleted == true || usr.ID != c.Sender().ID {
		return c.Send(a.cfg.Messages.Bars.NotAuthorized)
	}

	if err := a.usersStorage.Delete(context.Background(), c.Sender().ID); err != nil {
		a.logger.Errorf("failed to Delete due error: %s", err)
		return c.Send(a.cfg.Messages.BotError)
	}

	return c.Send(a.cfg.Messages.Bars.SuccessfulLogout)
}

func (a *app) handlePtCommand(c tele.Context) error {
	usr, err := a.usersStorage.FindOne(context.Background(), c.Sender().ID)
	if err != nil {
		a.logger.Errorf("failed to FindOne due error: %s", err)
		return c.Send(a.cfg.Messages.BotError)
	}

	if usr.Deleted == true || usr.ID != c.Sender().ID {
		return c.Send(a.cfg.Messages.Bars.NotAuthorized)
	}

	if len(usr.ProgressTable.Tables) == 0 {
		return c.Send(a.cfg.Messages.Bars.UnavailablePT)
	}

	progressTableInlineMarkup := makePtInlineMarkup(&usr.ProgressTable)

	msg := retrieveTablesData(usr.ProgressTable.Tables)
	return c.Send(msg, "Markdown", progressTableInlineMarkup)
}

func (a *app) handleGhCommand(c tele.Context) error {
	return c.Send("Github репозиторий бота: [ссылка](github.com/ilyadubrovsky/tracking-bars).", "Markdown")
}

func (a *app) handleText(c tele.Context) error {
	return c.Send(a.cfg.Messages.Bars.DefaultAnswer)
}

func (a *app) handleEchoCommand(c tele.Context) error {
	msg := strings.Replace(c.Message().Text, "/echo ", "", -1)
	return c.Send(msg, "Markdown")
}

func (a *app) handleSendnewsCommand(c tele.Context) error {
	msg := strings.Replace(c.Message().Text, "/sendnews ", "", -1)
	users, err := a.usersStorage.FindAll(context.Background())
	if err != nil {
		a.logger.Errorf("failed to FindAll due error: %s", err)
		return c.Send(a.cfg.Messages.BotError)
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
		return c.Send(a.cfg.Messages.BotError)
	}

	text := strings.Join(msg[2:], " ")
	_, err = a.bot.Send(tele.ChatID(userID), text)
	if err != nil {
		return err
	}
	return c.Send(fmt.Sprintf("Пользователю %d успешно отправлено сообщение:\n %s", userID, text))
}

func (a *app) handleDeluserCommand(c tele.Context) error {
	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 2 {
		return c.Send("Команда не содержит ID удаляемого пользователя.")
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		return c.Send(a.cfg.Messages.BotError)
	}

	if err = a.usersStorage.Delete(context.Background(), int64(userID)); err != nil {
		a.logger.Errorf("failed to Delete due error: %s", err)
		return c.Send(a.cfg.Messages.BotError)
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
	backButton := tele.InlineButton{
		Unique: "ptback",
		Text:   "Назад",
	}
	keyboard := make([][]tele.InlineButton, 1)
	keyboard[0] = append(keyboard[0], backButton)
	return &tele.ReplyMarkup{
		InlineKeyboard: keyboard,
	}
}
