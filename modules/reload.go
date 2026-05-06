package modules

import (
	"fmt"
	"html"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "reload",
		ModName: "🔄 Reload",
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

	x, _ := m.Reply(
		"⏳ <b>Refreshing Admin Cache...</b>",
		&telegram.SendOptions{ParseMode: "HTML"},
	)

	err := utils.RefreshAdminCache(m.Client, m.ChatID())
	if err != nil {
		_, err = x.Edit(fmt.Sprintf(
			"⚠️ <b>Cache Refresh Failed</b>\n\n"+
				"<i>I encountered an error while updating the administrator list.</i>\n\n"+
				"<blockquote><code>%s</code></blockquote>\n\n"+
				"💡 <b>Tip:</b> Please try again. If it fails repeatedly, ensure I have the <i>Add Admins</i> or <i>Manage Chat</i> permission.", html.EscapeString(err.Error()),
		),
			&telegram.SendOptions{ParseMode: "HTML"})
		return err
	}

	var text string
	if isb, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID()); isb {
		text = "✅ <b>Cache Updated!</b>\n\n" +
			"<i>The administrator list for this chat has been successfully synchronized.</i>\n\n" +
			"💡 <b>Tip:</b> New permissions or added admins should now be recognized."
	} else {
		text = "⚠️ <b>Cache Refreshed</b>\n\n" +
			"<i>I've updated the administrator list, but you still don't appear to be an admin.</i>\n\n" +
			"💡 <b>Tip:</b> If you were just promoted, wait a moment or try sending a message as an admin first."
	}

	_, err = x.Edit(text, &telegram.SendOptions{ParseMode: "HTML"})
	return err
}
