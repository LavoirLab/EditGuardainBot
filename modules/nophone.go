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
		Help: `<emoji id='5318779098686826724'>📞</emoji> <b>ᴘʜᴏɴᴇ ɴᴜᴍʙᴇʀ ᴘʀᴏᴛᴇᴄᴛɪᴏɴ</b> ʙʟᴏᴄᴋs ᴍᴇssᴀɢᴇs ᴄᴏɴᴛᴀɪɴɪɴɢ ᴘʜᴏɴᴇ ɴᴜᴍʙᴇʀs.

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/nophone on</code> — ʙʟᴏᴄᴋ ᴘʜᴏɴᴇ ɴᴜᴍʙᴇʀs <emoji id='6084551628461967966'>✅</emoji>
• <code>/nophone off</code> — ᴀʟʟᴏᴡ ᴘʜᴏɴᴇ ɴᴜᴍʙᴇʀs <emoji id='5465665476971471368'>❌</emoji></blockquote>

<b><emoji id='5420323339723881652'>ℹ️</emoji> ɴᴏᴛᴇ:</b>
<blockquote>– ᴍᴇssᴀɢᴇs ᴡɪᴛʜ ᴘʜᴏɴᴇ ɴᴜᴍʙᴇʀs ᴡɪʟʟ ʙᴇ ᴅᴇʟᴇᴛᴇᴅ.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nophone",
				Handler: RequirePermission("delete_messages", NoPhoneCmd),
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
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/nophone off</code> to allow phone numbers again."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/nophone on</code> to block phone numbers."
	}
	_, err = m.Reply(
		fmt.Sprintf("📞 <b>Phone number protection %s</b>\n\n%s", status, next),
	)
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
