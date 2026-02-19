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

	return nil
}
