package telegram

import "fmt"

type editableMessage struct {
	messageID int
	chatID    int64
}

func (e *editableMessage) MessageSig() (string, int64) {
	return fmt.Sprint(e.messageID), e.chatID
}
