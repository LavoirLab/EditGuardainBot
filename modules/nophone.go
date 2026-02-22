package modules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nophone",
		ModName: "📞 Phone",
		Help: "📞 <b>Phone Number Protection</b> blocks messages containing phone numbers.\n\n" +
			"<b>Commands:</b>\n" +
			"• <code>/nophone on</code> — Block phone numbers ✅\n" +
			"• <code>/nophone off</code> — Allow phone numbers ❌\n\n" +
			"<b>Detection:</b>\n" +
			"• International format: +91 9876543210\n" +
			"• With spaces/dashes: +1-234-567-8900\n" +
			"• Without plus: 919876543210\n\n" +
			"<b>ℹ️ Note:</b> Messages with phone numbers will be deleted.\n" +
			"👮 Only group admins can configure this setting.",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nophone",
				Handler: RequirePermission("change_info", NoPhoneCmd),
			},
		},
	})
}

var phoneRegex = regexp.MustCompile(
	`(?:(?:\+|00)[1-9]\d{0,3}[\s.-]?)?\(?\d{1,4}\)?[\s.-]?\d{1,4}[\s.-]?\d{1,9}`,
)

func NoPhoneCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, err := database.IsNoPhoneEnabled(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/nophone on</code> — Block phone numbers\n• <code>/nophone off</code> — Allow phone numbers",
			)
			return err
		} else {
			status := "Disabled ❌"
			if isEnabled {
				status = "Enabled ✅"
			}
			_, err = m.Reply(fmt.Sprintf("📞 <b>Phone Number Protection Status:</b> %s\n\n<i>When enabled, messages containing phone numbers will be automatically removed for privacy.</i>", status))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nophone on</code> or <code>/nophone off</code>.",
		)
		return err
	}

	if err := database.SetNoPhoneEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply("⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.")
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/nophone off</code> to allow phone numbers again."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/nophone on</code> to block phone numbers."
	}
	_, err = m.Reply(fmt.Sprintf("📞 <b>Phone number protection %s</b>\n\n%s", status, next))
	return err
}

func NoPhoneHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
		return nil
	}

	isEnabled, err := database.IsNoPhoneEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	if m.Text() == "" {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	// Check for phone numbers
	if !containsPhoneNumber(m.Text()) {
		return nil
	}

	// Delete message with phone number
	if _, err := m.Delete(); err != nil {
		return err
	}

	mention := utils.MentionHTML(m.Sender)
	_, _ = m.Reply(
		fmt.Sprintf(
			"📞 <b>Privacy Protection!</b>\n\n%s, <i>sharing phone numbers is not allowed here for safety and privacy reasons.</i> 🚫",
			mention,
		),
	)
	return telegram.ErrEndGroup
}

func containsPhoneNumber(text string) bool {
	matches := phoneRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return false
	}

	for _, match := range matches {
		digits := regexp.MustCompile(`\d`).FindAllString(match, -1)
		digitCount := len(digits)
		if digitCount >= 10 && digitCount <= 15 {
			return true
		}
	}

	return false
}
