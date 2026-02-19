package modules

import (
	"fmt"
	"html"
	"log"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
	"main/database"
)

func init() {
	AddHelp(
		"🚫 Abuse",
		"noabuse_help",
		"<b>🚫 NoAbuse Filter</b>\n"+
			"Automatically detects and filters abusive or offensive language in group messages.\n\n"+
			"<b>🔧 Commands:</b>\n"+
			"• <code>/noabuse on</code> – Enable abuse detection ✅\n"+
			"• <code>/noabuse off</code> – Disable abuse detection ❌\n\n"+
			"<b>ℹ️ Notes:</b>\n"+
			"– Messages with offensive content will be censored or removed.\n"+
			"– 👮 Only group admins can configure this setting.\n\n"+
			"<blockquote>🔧 This feature is under development and may not work properly at this moment.</blockquote>",
	)
}

func NoAbuseCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Text())
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> abuse -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> abuse -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")

		return telegram.EndGroup
	}

	if len(args) < 2 {
		m.Respond("⚠️ Usage:\n<code>/noabuse on</code> – Enable Abuse detection\n<code>/noabuse off</code> – Disable abuse detection")
		return telegram.EndGroup
	}
	arg := strings.ToLower(args[1])
	enable := arg == "on"
	if arg != "on" && arg != "off" {
		m.Respond("❌ Invalid option.\nUse <code>/noabuse on</code> or <code>/noabuse off</code>")
		return telegram.EndGroup
	}
	if err := database.SetNSFWFlag(m.ChatID(), enable); err != nil {
		log.Println("NoAbuse.error:", err)
		m.Respond("❌ Failed to update setting.")
		return telegram.EndGroup
	}
	status := "🛡️ NoAbuse detection is enabled ✅"
	if !enable {
		status = "🚫 NoAbuse detection is disabled ❌"
	}
	m.Respond(status)
	return telegram.EndGroup
}

func DeleteAbuseHandle(m *telegram.NewMessage) error {
	if bo := IsSupergroup(m); !bo {
		return nil
	}

	if !database.IsNSFWEnabled(m.ChatID()) {
		return nil
	}
	if ShouldIgnoreGroupAnonymous(m) {
		return nil
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.Sender.ID); err != nil {
		L(m, "Modules -> abuse -> utils.IsChatAdmin()", err)
		return nil
	} else if isadmin {
		return nil
	}
	isNsfw := utils.ContainsAbusiveWord(m.Text())
	if !isNsfw {
		return nil
	}

	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> abuse -> m.Delete()", err)
	}
	var user string
	if m.Sender.Username != "" {
		user = "@" + m.Sender.Username
	} else {
		userFullName := strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		user = fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, m.SenderID(), html.EscapeString(userFullName))
	}
	m.Respond(fmt.Sprintf("🚫 %s, Your message was deleted due to abusive words.", user))

	return telegram.EndGroup
}
