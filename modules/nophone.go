package modules

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func getNoPhoneDashboard(enabled bool) (string, telegram.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("📞 <b>Phone Protection</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n\n"+
		"<blockquote expandable>"+
		"When enabled, I automatically detect and remove messages containing phone numbers to protect user privacy and prevent unwanted solicitations.\n\n"+
		"<b>Action:</b> Phone numbers are detected using advanced pattern matching and removed instantly to maintain a secure environment."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> Use <code>/nophone on</code> or <code>/nophone off</code> to quickly toggle this setting.", status)

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data(toggleLabel, "nophone_toggle:"+toggleVal),
		telegram.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard
}

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
			{
				Pattern: "callback:^nophone_toggle:(on|off)$",
				Handler: handleNoPhoneToggle,
			},
		},
	})
}

func handleNoPhoneToggle(c *telegram.CallbackQuery) error {
	if isAdmin, _ := utils.IsAdmin(c.Client, c.ChatID, c.SenderID); !isAdmin {
		c.Answer(
			"👤 Admin Access Required",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	if err := database.SetNoPhoneEnabled(c.ChatID, enable); err != nil {
		c.Answer("⚠️ Update failed.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	text, keyboard := getNoPhoneDashboard(enable)
	c.Answer("✓ Settings Updated")
	_, err := c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
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
		isEnabled, _ := database.IsNoPhoneEnabled(m.ChatID())
		text, keyboard := getNoPhoneDashboard(isEnabled)
		_, err = m.Reply(text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nophone on</code> or <code>/nophone off</code>.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	if err := database.SetNoPhoneEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	text, keyboard := getNoPhoneDashboard(enable)
	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NoPhoneHandler(m *telegram.NewMessage) error {
	// Fast path: check for entities before any other processing
	if len(m.Message.Entities) == 0 || m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	// Optimization: Fast path check for phone entities before any expensive operations.
	// This avoids redundant database lookups and admin checks for messages without phone numbers.
	hasPhone := false
	for _, entity := range m.Message.Entities {
		if _, ok := entity.(*telegram.MessageEntityPhone); ok {
			hasPhone = true
			break
		}
	}

	if !hasPhone {
		return nil
	}

	isEnabled, err := database.IsNoPhoneEnabled(m.ChatID())
	if err != nil || !isEnabled {
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
	mm, _ := m.Reply(
		fmt.Sprintf(
			"📞 <b>Privacy Protection!</b>\n\n%s, <i>sharing phone numbers is not allowed here for safety and privacy reasons.</i> 🚫",
			mention,
		),
	)
	if mm != nil {
		ScheduleDeletion(mm, 30*time.Second)
	}
	return telegram.ErrEndGroup
}

func containsPhoneNumber(text string) bool {
	if !phoneRegex.MatchString(text) {
		return false
	}

	matches := phoneRegex.FindAllStringIndex(text, -1)
	for _, match := range matches {
		digitCount := 0
		for i := match[0]; i < match[1]; i++ {
			if text[i] >= '0' && text[i] <= '9' {
				digitCount++
			}
		}
		if digitCount >= 10 && digitCount <= 15 {
			return true
		}
	}

	return false
}
