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
				Group:   -1, // Early group for serving
			},
		},
	})
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

	handlers := []func(*telegram.NewMessage) error{
		AutoDeleteHandler,
		MediaDeleteHandler,
		AbuseHandler,
		NoForwardHandler,
		NoHashtagsHandler,
		NoPhoneHandler,
		LinkFilterHandler,
		NoPromoHandler,
		LongMessageHandler,
		BioLinkHandler,
		NsfwHandler,
	}

	for _, handler := range handlers {
		if err := handler(m); err != nil {
			return err
		}
	}

	return nil
}
