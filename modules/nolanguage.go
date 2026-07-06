package modules

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func getNoLanguageDashboard(enabled bool) (string, telegram.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("🌐 <b>Language Filter</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n\n"+
		"<blockquote expandable>"+
		"When enabled, only messages written in English or Hindi are allowed.\n\n"+
		"<b>Action:</b> Messages containing any other script/language are deleted, a warning is sent, and both are auto-cleared after 30 seconds."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> Use <code>/nolanguage on</code> or <code>/nolanguage off</code> to quickly toggle this setting.", status)

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data(toggleLabel, "nolanguage_toggle:"+toggleVal),
		telegram.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard
}

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nolanguage",
		ModName: "🌐 Language",
		Help: `<emoji id='5431343731163611116'>🌐</emoji> <b>ʟᴀɴɢᴜᴀɢᴇ ғɪʟᴛᴇʀ</b> ᴀʟʟᴏᴡs ᴏɴʟʏ ᴇɴɢʟɪsʜ ᴀɴᴅ ʜɪɴᴅɪ ᴛᴇxᴛ.

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/nolanguage on</code> — ʙʟᴏᴄᴋ ᴏᴛʜᴇʀ ʟᴀɴɢᴜᴀɢᴇs <emoji id='6084551628461967966'>✅</emoji>
• <code>/nolanguage off</code> — ᴀʟʟᴏᴡ ᴀʟʟ ʟᴀɴɢᴜᴀɢᴇs <emoji id='5465665476971471368'>❌</emoji></blockquote>

<b><emoji id='5447644880824181073'>⚠️</emoji> ᴀᴄᴛɪᴏɴ:</b>
<blockquote>– ᴍᴇssᴀɢᴇs ɪɴ ᴀɴʏ ᴏᴛʜᴇʀ sᴄʀɪᴘᴛ ᴀʀᴇ ᴅᴇʟᴇᴛᴇᴅ ɪᴍᴍᴇᴅɪᴀᴛᴇʟʏ.
– ᴀ ᴡᴀʀɴɪɴɢ ɪs sᴇɴᴛ ᴀɴᴅ ᴀᴜᴛᴏ-ᴅᴇʟᴇᴛᴇᴅ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nolanguage",
				Handler: RequirePermission("delete_messages", NoLanguageCmd),
			},
			{
				Pattern: "callback:^nolanguage_toggle:(on|off)$",
				Handler: handleNoLanguageToggle,
			},
		},
	})
}

func handleNoLanguageToggle(c *telegram.CallbackQuery) error {
	if isAdmin, _ := utils.IsAdmin(c.Client, c.ChatID, c.SenderID); !isAdmin {
		c.Answer(
			"👤 Admin Access Required",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	if err := database.SetNoLanguageEnabled(c.ChatID, enable); err != nil {
		c.Answer("⚠️ Update failed.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	text, keyboard := getNoLanguageDashboard(enable)
	c.Answer("✓ Settings Updated")
	_, err := c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NoLanguageCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, _ := database.IsNoLanguageEnabled(m.ChatID())
		text, keyboard := getNoLanguageDashboard(isEnabled)
		_, err = m.Reply(text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nolanguage on</code> or <code>/nolanguage off</code>.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	if err := database.SetNoLanguageEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	text, keyboard := getNoLanguageDashboard(enable)
	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NoLanguageHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	text := m.Text()
	if text == "" {
		return nil
	}

	isEnabled, err := database.IsNoLanguageEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	if !containsForeignScript(text) {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	mention := utils.MentionHTML(m.Sender)
	mm, _ := m.Reply(
		fmt.Sprintf(
			"🌐 <b>Language Not Allowed!</b>\n\n%s, <i>only English and Hindi are permitted in this group.</i> 🚫",
			mention,
		),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	if mm != nil {
		ScheduleDeletion(mm, 30*time.Second)
	}
	return telegram.ErrEndGroup
}

// containsForeignScript reports whether text contains any character outside
// English (Latin script), Hindi (Devanagari script), and script-neutral
// characters (digits, punctuation, symbols, emoji, whitespace, combining marks).
func containsForeignScript(text string) bool {
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		if unicode.In(
			r,
			unicode.Latin,
			unicode.Devanagari,
			unicode.Common,
			unicode.Inherited,
		) {
			continue
		}
		return true
	}
	return false
}
