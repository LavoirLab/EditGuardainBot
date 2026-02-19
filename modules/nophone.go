// file: modules/nophone.go
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
		"📞 NoPhone",
		"nophone_help",
		"📞 <b>Phone Number Protection</b> blocks messages containing phone numbers.\n\n"+
			"<b>Commands:</b>\n"+
			"• <code>/nophone on</code> — Block phone numbers ✅\n"+
			"• <code>/nophone off</code> — Allow phone numbers ❌\n\n"+
			"<b>Detection:</b>\n"+
			"• International format: +91 9876543210\n"+
			"• With spaces/dashes: +1-234-567-8900\n"+
			"• Without plus: 919876543210\n\n"+
			"<b>ℹ️ Note:</b> Messages with phone numbers will be deleted.\n"+
			"👮 Only group admins can configure this setting.",
	)
}

var phoneRegex = regexp.MustCompile(`(?:(?:\+|00)[1-9]\d{0,3}[\s.-]?)?\(?\d{1,4}\)?[\s.-]?\d{1,4}[\s.-]?\d{1,9}`)

func NoPhoneCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Text())
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nophone -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> nophone -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")
		return telegram.EndGroup
	}

	if len(args) < 2 {
		isEnabled, err := database.IsNoPhoneEnabled(m.ChatID())
		if err != nil {
			m.Respond("⚠️ Usage:\n<code>/nophone on</code> — Enable\n<code>/nophone off</code> — Disable")
		} else {
			status := map[bool]string{true: "Enabled", false: "Disabled"}[isEnabled]
			m.Respond(fmt.Sprintf("Currently NoPhone is %s for your chat.", status))
		}
		return telegram.EndGroup
	}

	arg := strings.ToLower(args[1])
	enable := arg == "on"
	if arg != "on" && arg != "off" {
		m.Respond("❌ Invalid option.\nUse <code>/nophone on</code> or <code>/nophone off</code>")
		return telegram.EndGroup
	}

	if err := database.SetNoPhoneEnabled(m.ChatID(), enable); err != nil {
		m.Respond("❌ Failed to update setting.")
		return L(m, "Modules -> nophone -> SetNoPhoneEnabled", err)
	}

	status := "📞 Phone number blocking enabled ✅"
	if !enable {
		status = "✅ Phone numbers allowed ❌"
	}
	m.Respond(status)
	return telegram.EndGroup
}

func handlePhoneNumber(m *telegram.NewMessage) error {
	if !IsSupergroup(m) {
		return nil
	}

	isEnabled, err := database.IsNoPhoneEnabled(m.ChatID())
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
		L(m, "Modules -> nophone -> utils.IsChatAdmin()", err)
		return nil
	}
	if isadmin {
		return nil
	}

	// Check for phone numbers
	if !containsPhoneNumber(m.Text()) {
		return nil
	}

	// Delete message with phone number
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nophone -> m.Delete()", err)
	}

	var mention string
	if m.Sender.Username != "" {
		mention = "@" + m.Sender.Username
	} else {
		mention = fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", m.Sender.ID, html.EscapeString(m.Sender.FirstName))
	}

	m.Respond(fmt.Sprintf("🚫 %s, sharing phone numbers is not allowed in this group.", mention))
	return telegram.EndGroup
}

func containsPhoneNumber(text string) bool {
	matches := phoneRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return false
	}

	// Filter out false positives (numbers that are too short or clearly not phone numbers)
	for _, match := range matches {
		// Remove all non-digit characters
		digits := regexp.MustCompile(`\d`).FindAllString(match, -1)
		digitCount := len(digits)

		// Valid phone numbers typically have 10-15 digits
		if digitCount >= 10 && digitCount <= 15 {
			return true
		}
	}

	return false
}
