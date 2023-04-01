package grades

import (
	"encoding/json"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"regexp"
	"strconv"
	"strings"
	"telegram-service/internal/entity/user"
	"telegram-service/internal/events/model"
)

type ProcessStrategy struct {
	service       model.Service
	botError      string
	unavailablePT string
}

func (s *ProcessStrategy) Process(body []byte) error {
	var response GetGradesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("json unmarshal: %v", err)
	}

	if response.ResponseMessage != "" {
		if !response.IsCallback {
			return s.service.SendMessageWithOpts(response.RequestID, response.ResponseMessage)
		}
		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID,
			response.ResponseMessage)
	}

	if !response.IsCallback {
		return s.service.SendMessageWithOpts(response.RequestID, extractTablesData(response.ProgressTable.Tables),
			tele.ModeMarkdown, makePtInlineMarkup(&response.ProgressTable))
	}

	if response.CallbackData == "back" {
		progressTableInlineMarkup := makePtInlineMarkup(&response.ProgressTable)
		msg := extractTablesData(response.ProgressTable.Tables)
		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, msg, tele.ModeMarkdown, progressTableInlineMarkup)
	} else if strings.HasPrefix(response.CallbackData, "show") {
		n, err := strconv.Atoi(response.CallbackData[4:])
		if err != nil {
			return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.botError)
		}

		if n <= 0 || n > len(response.ProgressTable.Tables) {
			return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.unavailablePT)
		}

		ptBackInlineMarkup := makePtBackInlineMarkupWithHide(n)

		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID,
			response.ProgressTable.Tables[n-1].String(), tele.ModeMarkdown, ptBackInlineMarkup)
	} else if regexp.MustCompile(`^[0-9]+$`).MatchString(response.CallbackData) {
		n, err := strconv.Atoi(response.CallbackData)
		if err != nil {
			return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.botError)
		}

		if n <= 0 || n > len(response.ProgressTable.Tables) {
			return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.unavailablePT)
		}

		ptBackInlineMarkup := makePtBackInlineMarkupWithShow(n)

		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID,
			hideStNames(&response.ProgressTable.Tables[n-1]).String(), tele.ModeMarkdown, ptBackInlineMarkup)
	} else {
		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.botError)
	}
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

// TODO optimize code

func makePtBackInlineMarkupWithHide(numberSubject int) *tele.ReplyMarkup {
	hideButton := tele.InlineButton{
		Unique: fmt.Sprintf("pt%d", numberSubject),
		Text:   "Скрыть названия КМов",
	}

	backButton := tele.InlineButton{
		Unique: "ptback",
		Text:   "Назад",
	}
	keyboard := make([][]tele.InlineButton, 2)
	keyboard[0] = append(keyboard[0], hideButton)
	keyboard[1] = append(keyboard[1], backButton)
	return &tele.ReplyMarkup{
		InlineKeyboard: keyboard,
	}
}

func makePtBackInlineMarkupWithShow(numberSubject int) *tele.ReplyMarkup {
	showButton := tele.InlineButton{
		Unique: fmt.Sprintf("ptshow%d", numberSubject),
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

func extractTablesData(tables []user.SubjectTable) string {
	var result string
	for i, subjectTable := range tables {
		result += fmt.Sprintf("*%d:* %s\n\n", i+1, subjectTable.Name)
	}
	result += "Для просмотра оценок по определённому предмету, пользуйтесь кнопочным меню."
	return result
}

func NewProcessStrategy(service model.Service, botError, unavailablePT string) *ProcessStrategy {
	return &ProcessStrategy{service: service, botError: botError, unavailablePT: unavailablePT}
}
