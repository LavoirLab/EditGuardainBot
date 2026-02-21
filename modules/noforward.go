package modules

import (
	"fmt"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "noforward",
		ModName: "📤 NoForward",
		Help: "📤 <b>Forward Control</b> blocks forwarded messages in the group.\n\n" +
			"<b>Commands:</b>\n" +
			"• <code>/noforward on</code> — Block forwarded messages ✅\n" +
			"• <code>/noforward off</code> — Allow forwarded messages ❌\n\n" +
			"<b>ℹ️ Note:</b> Forwarded messages will be automatically deleted.\n" +
			"👮 Only group admins can configure this setting.",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:noforward",
				Handler: RequirePermission("change_info", NoForwardCmd),
			},
		},
	})
}

func NoForwardCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, err := database.IsNoForwardEnabled(m.ChatID())
		if err != nil {
			_, err = m.Reply("⚠️ Usage:\n<code>/noforward on</code> — Enable\n<code>/noforward off</code> — Disable")
			return err
		} else {
			status := map[bool]string{true: "Enabled", false: "Disabled"}[isEnabled]
			_, err = m.Reply(fmt.Sprintf("Currently NoForward is %s for your chat.", status))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply("❌ Invalid option.\nUse <code>/noforward on</code> or <code>/noforward off</code>")
		return err
	}

	if err := database.SetNoForwardEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply("❌ Failed to update setting.")
		return err
	}

	status := "📤 Forward blocking enabled ✅"
	if !enable {
		status = "✅ Forwarding allowed ❌"
	}
	_, err = m.Reply(status)
	return err
}

func NoForwardHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
		return nil
	}

	isEnabled, err := database.IsNoForwardEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	// Check if message is forwarded
	if m.Message.FwdFrom == nil {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	// Delete forwarded message
	if _, err := m.Delete(); err != nil {
		return err
	}

	mention := utils.MentionHTML(m.Sender)
	_, _ = m.Reply(fmt.Sprintf("📤 <b>NoForward Enabled!</b>\n\n%s, <i>forwarding messages is restricted in this group to maintain original conversations.</i> 🚫", mention))
	return telegram.ErrEndGroup
}
