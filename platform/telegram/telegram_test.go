package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestCallbackCoreMessageMarksPrivateChatAsDM(t *testing.T) {
	p := &Platform{}
	cb := &tgbotapi.CallbackQuery{
		Data: "cmd:/project switch repo-a",
		From: &tgbotapi.User{
			ID:       123,
			UserName: "tester",
		},
		Message: &tgbotapi.Message{
			MessageID: 9,
			Chat: &tgbotapi.Chat{
				ID:   456,
				Type: "private",
			},
		},
	}

	msg := p.callbackCoreMessage(cb, "/project switch repo-a")
	if !msg.IsDM {
		t.Fatal("expected callback message from private chat to be marked as DM")
	}
}
