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
		ModName: "📤 Forward",
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
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/noforward on</code> — Block forwarded messages\n• <code>/noforward off</code> — Allow forwarded messages",
			)
			return err
		} else {
			status := "Disabled ❌"
			if isEnabled {
				status = "Enabled ✅"
			}
			_, err = m.Reply(fmt.Sprintf("📤 <b>Forward Control Status:</b> %s\n\n<i>When enabled, forwarded messages from other chats will be automatically removed.</i>", status))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/noforward on</code> or <code>/noforward off</code>.",
		)
		return err
	}

	if err := database.SetNoForwardEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/noforward off</code> to allow forwarded messages again."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/noforward on</code> to block forwarded messages."
	}
	_, err = m.Reply(
		fmt.Sprintf("📤 <b>Forward control %s</b>\n\n%s", status, next),
	)
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
	_, _ = m.Reply(
		fmt.Sprintf(
			"📤 <b>NoForward Enabled!</b>\n\n%s, <i>forwarding messages is restricted in this group to maintain original conversations.</i> 🚫",
			mention,
		),
	)
	return telegram.ErrEndGroup
}
