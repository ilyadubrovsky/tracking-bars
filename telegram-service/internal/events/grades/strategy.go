package grades

import (
	"encoding/json"
	"fmt"
	"github.com/ilyadubrovsky/bars"
	tele "gopkg.in/telebot.v3"
	"regexp"
	"strconv"
	"strings"
)

type messageProvider interface {
	EditMessageWithOpts(id int64, messageid int, msg string, opts ...interface{}) error
	SendMessageWithOpts(id int64, msg string, opts ...interface{}) error
}

type ProcessStrategy struct {
	service       messageProvider
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
		return s.service.SendMessageWithOpts(response.RequestID, extractSubjectsNames(response.ProgressTable.Tables),
			tele.ModeMarkdown, makePtInlineMarkup(&response.ProgressTable))
	}

	if response.CallbackData == "back" {
		progressTableInlineMarkup := makePtInlineMarkup(&response.ProgressTable)
		msg := extractSubjectsNames(response.ProgressTable.Tables)
		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, msg, tele.ModeMarkdown, progressTableInlineMarkup)
	} else if strings.HasPrefix(response.CallbackData, "show") {
		n, err := strconv.Atoi(response.CallbackData[4:])
		if err != nil {
			return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.botError)
		}

		if n <= 0 || n > len(response.ProgressTable.Tables) {
			return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.unavailablePT)
		}

		ptBackInlineMarkup := makePtBackInlineMarkup(n, "hide")

		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID,
			stToTelegramMessage(&response.ProgressTable.Tables[n-1]), tele.ModeMarkdown, ptBackInlineMarkup)
	} else if regexp.MustCompile(`^[0-9]+$`).MatchString(response.CallbackData) {
		n, err := strconv.Atoi(response.CallbackData)
		if err != nil {
			return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.botError)
		}

		if n <= 0 || n > len(response.ProgressTable.Tables) {
			return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.unavailablePT)
		}

		ptBackInlineMarkup := makePtBackInlineMarkup(n, "show")

		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID,
			stToTelegramMessage(hideControlEventsNames(&response.ProgressTable.Tables[n-1])), tele.ModeMarkdown, ptBackInlineMarkup)
	} else {
		return s.service.EditMessageWithOpts(response.RequestID, response.MessageID, s.botError)
	}
}

func stToTelegramMessage(st *bars.SubjectTable) string {
	var b strings.Builder

	fmt.Fprintf(&b, fmt.Sprintf("*Название дисциплины:*\n%s\n\n", st.Name))

	for _, c := range st.ControlEvents {
		fmt.Fprintf(&b, fmt.Sprintf("%s\n*Оценка:* %s\n\n", c.Name, c.Grades))
	}

	return b.String()
}

func makePtInlineMarkup(pt *bars.ProgressTable) *tele.ReplyMarkup {

	numberOfRows, numberOfButtonsInLastRow, numberOfButtonsInRow := 0, 0, 5
	if numberOfSubjects := len(pt.Tables); numberOfSubjects >= numberOfButtonsInRow {
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

// makePtBackInlineMarkup showOrHide must be "show" or "hide", its type of button in PtInlineMarkup
func makePtBackInlineMarkup(numberSubject int, showOrHide string) *tele.ReplyMarkup {
	backButton := tele.InlineButton{
		Unique: "ptback",
		Text:   "Назад",
	}

	var HideOrShowButton tele.InlineButton
	if showOrHide == "show" {
		HideOrShowButton = tele.InlineButton{
			Unique: fmt.Sprintf("ptshow%d", numberSubject),
			Text:   "Показать названия КМов",
		}
	} else {
		HideOrShowButton = tele.InlineButton{
			Unique: fmt.Sprintf("pt%d", numberSubject),
			Text:   "Скрыть названия КМов",
		}
	}

	keyboard := make([][]tele.InlineButton, 2)
	keyboard[0] = append(keyboard[0], HideOrShowButton)
	keyboard[1] = append(keyboard[1], backButton)

	return &tele.ReplyMarkup{
		InlineKeyboard: keyboard,
	}
}

func hideControlEventsNames(st *bars.SubjectTable) *bars.SubjectTable {
	for i := range st.ControlEvents {
		if !(strings.HasPrefix(st.ControlEvents[i].Name, "Балл текущего контроля") ||
			strings.HasPrefix(st.ControlEvents[i].Name, "Итоговая оценка:") ||
			strings.HasPrefix(st.ControlEvents[i].Name, "Промежуточная аттестация")) {
			st.ControlEvents[i].Name = fmt.Sprintf("КМ-%d", i+1)
		}
	}
	return st
}

func extractSubjectsNames(tables []bars.SubjectTable) string {
	var b strings.Builder

	for i, st := range tables {
		fmt.Fprintf(&b, fmt.Sprintf("*%d:* %s\n\n", i+1, st.Name))
	}

	fmt.Fprint(&b, "Для просмотра оценок по определённому предмету, пользуйтесь кнопочным меню.")

	return b.String()
}

func NewProcessStrategy(service messageProvider, botError, unavailablePT string) *ProcessStrategy {
	return &ProcessStrategy{service: service, botError: botError, unavailablePT: unavailablePT}
}
