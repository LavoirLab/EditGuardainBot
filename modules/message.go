package modules

import (
	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "message",
		ModName: "Message",
		Handlers: []HandlerConfig{
			{
				Pattern: "message",
				Handler: OnMessageFnc,
			},
		},
	})
}

var messageProtectionHandlers = []func(*telegram.NewMessage) error{
	AutoDeleteHandler,
	MediaDeleteHandler,
	AbuseHandler,
	NoForwardHandler,
	NoHashtagsHandler,
	NoPhoneHandler,
	LinkFilterHandler,
	NoPromoHandler,
	StickerBlockHandler,
	LongMessageHandler,
	BioLinkHandler,
	NsfwHandler,
}

func OnMessageFnc(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		database.AddServedUser(m.ChatID())
	} else if m.IsGroup() {
		database.AddServedChat(m.ChatID())
	}

	if !m.IsGroup() || m.IsChannelPost() {
		return nil
	}

	for _, handler := range messageProtectionHandlers {
		if err := handler(m); err != nil {
			return err
		}
	}

	return nil
}
