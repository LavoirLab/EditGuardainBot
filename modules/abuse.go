package modules

import (
	"fmt"
	"log"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "abuse",
		ModName: "🚫 Abuse",
		Help: "<b>🚫 NoAbuse Filter</b>\n" +
			"Automatically detects and filters abusive or offensive language in group messages.\n\n" +
			"<b>🔧 Commands:</b>\n" +
			"• <code>/noabuse on</code> – Enable abuse detection ✅\n" +
			"• <code>/noabuse off</code> – Disable abuse detection ❌\n\n" +
			"<b>ℹ️ Notes:</b>\n" +
			"– Messages with offensive content will be censored or removed.\n" +
			"– 👮 Only group admins can configure this setting.\n\n" +
			"<blockquote>🔧 This feature is under development and may not work properly at this moment.</blockquote>",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:noabuse",
				Handler: RequirePermission("change_info", NoAbuseCmd),
			},
			{
				Pattern: "message",
				Handler: DeleteAbuseHandle,
			},
		},
	})
}

func NoAbuseCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Args())
	if _, err := m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		_, err := m.Reply("⚠️ Usage:\n<code>/noabuse on</code> – Enable Abuse detection\n<code>/noabuse off</code> – Disable abuse detection")
		return err
	}
	arg := strings.ToLower(args[0])
	enable := arg == "on"
	if arg != "on" && arg != "off" {
		_, err := m.Reply("❌ Invalid option.\nUse <code>/noabuse on</code> or <code>/noabuse off</code>")
		return err
	}
	if err := database.SetNSFWFlag(m.ChatID(), enable); err != nil {
		log.Println("NoAbuse.error:", err)
		_, err := m.Reply("❌ Failed to update setting.")
		return err
	}
	status := "🛡️ NoAbuse detection is enabled ✅"
	if !enable {
		status = "🚫 NoAbuse detection is disabled ❌"
	}
	_, err := m.Reply(status)
	return err
}

func DeleteAbuseHandle(m *telegram.NewMessage) error {
	if !m.IsGroup() || m.Channel == nil || m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	if !database.IsNSFWEnabled(m.ChatID()) {
		return nil
	}

	if isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID()); isadmin {
		return nil
	}

	isNsfw := utils.ContainsAbusiveWord(m.Text())
	if !isNsfw {
		return nil
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	user := utils.MentionHTML(m.Sender)
	_, err := m.Reply(fmt.Sprintf("🚫 %s, Your message was deleted due to abusive words.", user))

	return err
}
