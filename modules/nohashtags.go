package modules

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
	"main/database"
)

func init() {
	AddHelp(
		"# NoHashtags",
		"nohashtags_help",
		"# <b>Hashtag Filter</b> blocks messages containing hashtags.\n\n"+
			"<b>Commands:</b>\n"+
			"• <code>/nohashtags on</code> — Block hashtags ✅\n"+
			"• <code>/nohashtags off</code> — Allow hashtags ❌\n\n"+
			"<b>Detection:</b>\n"+
			"• Any word starting with # symbol\n"+
			"• Example: #join, #promotion, #trending\n\n"+
			"<b>ℹ️ Note:</b> Messages with hashtags will be deleted.\n"+
			"👮 Only group admins can configure this setting.",
	)
}

var hashtagRegex = regexp.MustCompile(`#\w+`)

func NoHashtagsCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Text())
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nohashtags -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> nohashtags -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")
		return telegram.EndGroup
	}

	if len(args) < 2 {
		isEnabled, err := database.IsNoHashtagsEnabled(m.ChatID())
		if err != nil {
			m.Respond("⚠️ Usage:\n<code>/nohashtags on</code> — Enable\n<code>/nohashtags off</code> — Disable")
		} else {
			status := map[bool]string{true: "Enabled", false: "Disabled"}[isEnabled]
			m.Respond(fmt.Sprintf("Currently NoHashtags is %s for your chat.", status))
		}
		return telegram.EndGroup
	}

	arg := strings.ToLower(args[1])
	enable := arg == "on"
	if arg != "on" && arg != "off" {
		m.Respond("❌ Invalid option.\nUse <code>/nohashtags on</code> or <code>/nohashtags off</code>")
		return telegram.EndGroup
	}

	if err := database.SetNoHashtagsEnabled(m.ChatID(), enable); err != nil {
		m.Respond("❌ Failed to update setting.")
		return L(m, "Modules -> nohashtags -> SetNoHashtagsEnabled", err)
	}

	status := "# Hashtag blocking enabled ✅"
	if !enable {
		status = "✅ Hashtags allowed ❌"
	}
	m.Respond(status)
	return telegram.EndGroup
}

func handleHashtags(m *telegram.NewMessage) error {
	if !IsSupergroup(m) {
		return nil
	}

	isEnabled, err := database.IsNoHashtagsEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	if m.Text() == "" {
		return nil
	}

	if ShouldIgnoreGroupAnonymous(m) {
		return nil
	}

	isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.Sender.ID)
	if err != nil {
		L(m, "Modules -> nohashtags -> utils.IsChatAdmin()", err)
		return nil
	}
	if isadmin {
		return nil
	}

	// Check for hashtags
	if !containsHashtag(m.Text()) {
		return nil
	}

	// Delete message with hashtags
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nohashtags -> m.Delete()", err)
	}

	var mention string
	if m.Sender.Username != "" {
		mention = "@" + m.Sender.Username
	} else {
		mention = fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", m.Sender.ID, html.EscapeString(m.Sender.FirstName))
	}

	m.Respond(fmt.Sprintf("🚫 %s, hashtags are not allowed in this group.", mention))
	return telegram.EndGroup
}

func containsHashtag(text string) bool {
	return hashtagRegex.MatchString(text)
}
