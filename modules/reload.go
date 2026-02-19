package modules

import (
	"fmt"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "reload",
		ModName: "Reload",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:reload",
				Handler: ReloadHandler,
			},
		},
	})
}

func ReloadHandler(m *telegram.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}
	if !m.IsGroup() || m.Channel == nil {
		return nil
	}

	x, _ := m.Reply("Refreshing Cache of chat admins...")

	err := utils.RefreshAdminCache(m.Client, m.ChatID())
	if err != nil {
		_, err = x.Edit(fmt.Sprintf("⚠️ Cache refresh failed — %v", err))
		return err
	}

	var text string
	if isb, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID()); isb {
		text = "✅ Successfully refreshed the cache of chat admins!"
	} else {
		text = "⚠️ Tried refreshing the admin cache... but it still seems you're not an admin!"
	}

	_, err = x.Edit(text)
	return err
}
