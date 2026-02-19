// file: modules/noforward.go
package modules

import (
	"fmt"
	"html"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
	"main/database"
)

func init() {
	AddHelp(
		"📤 NoForward",
		"noforward_help",
		"📤 <b>Forward Control</b> blocks forwarded messages in the group.\n\n"+
			"<b>Commands:</b>\n"+
			"• <code>/noforward on</code> — Block forwarded messages ✅\n"+
			"• <code>/noforward off</code> — Allow forwarded messages ❌\n\n"+
			"<b>ℹ️ Note:</b> Forwarded messages will be automatically deleted.\n"+
			"👮 Only group admins can configure this setting.",
	)
}

func NoForwardCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Text())
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> noforward -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> noforward -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")
		return telegram.EndGroup
	}

	if len(args) < 2 {
		isEnabled, err := database.IsNoForwardEnabled(m.ChatID())
		if err != nil {
			m.Respond("⚠️ Usage:\n<code>/noforward on</code> — Enable\n<code>/noforward off</code> — Disable")
		} else {
			status := map[bool]string{true: "Enabled", false: "Disabled"}[isEnabled]
			m.Respond(fmt.Sprintf("Currently NoForward is %s for your chat.", status))
		}
		return telegram.EndGroup
	}

	arg := strings.ToLower(args[1])
	enable := arg == "on"
	if arg != "on" && arg != "off" {
		m.Respond("❌ Invalid option.\nUse <code>/noforward on</code> or <code>/noforward off</code>")
		return telegram.EndGroup
	}

	if err := database.SetNoForwardEnabled(m.ChatID(), enable); err != nil {
		m.Respond("❌ Failed to update setting.")
		return L(m, "Modules -> noforward -> SetNoForwardEnabled", err)
	}

	status := "📤 Forward blocking enabled ✅"
	if !enable {
		status = "✅ Forwarding allowed ❌"
	}
	m.Respond(status)
	return telegram.EndGroup
}

func handleForwardedMessage(m *telegram.NewMessage) error {
	if !IsSupergroup(m) {
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

	if ShouldIgnoreGroupAnonymous(m) {
		return nil
	}

	isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.Sender.ID)
	if err != nil {
		L(m, "Modules -> noforward -> utils.IsChatAdmin()", err)
		return nil
	}
	if isadmin {
		return nil
	}

	// Delete forwarded message
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> noforward -> m.Delete()", err)
	}

	var mention string
	if m.Sender.Username != "" {
		mention = "@" + m.Sender.Username
	} else {
		mention = fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", m.Sender.ID, html.EscapeString(m.Sender.FirstName))
	}

	m.Respond(fmt.Sprintf("🚫 %s, forwarding messages is not allowed in this group.", mention))
	return telegram.EndGroup
}
