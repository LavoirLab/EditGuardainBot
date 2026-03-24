package modules

import (
	"fmt"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nostylish",
		ModName: "🔡 Fonts",
		Help: `<emoji id='5431343731163611116'>🔡</emoji> <b>sᴛʏʟɪsʜ ғᴏɴᴛ ғɪʟᴛᴇʀ</b>

ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇᴛᴇᴄᴛs ᴀɴᴅ ʀᴇᴍᴏᴠᴇs ᴍᴇssᴀɢᴇs sᴇɴᴛ ᴡɪᴛʜ sᴛʏʟɪsʜ ᴏʀ ᴅᴇᴄᴏʀᴀᴛɪᴠᴇ ғᴏɴᴛs.

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/nostylish on</code> — ᴇɴᴀʙʟᴇ sᴛʏʟɪsʜ ғᴏɴᴛ ғɪʟᴛᴇʀ <emoji id='6084551628461967966'>✅</emoji>
• <code>/nostylish off</code> — ᴅɪsᴀʙʟᴇ sᴛʏʟɪsʜ ғᴏɴᴛ ғɪʟᴛᴇʀ <emoji id='5465665476971471368'>❌</emoji></blockquote>

<b><emoji id='5447644880824181073'>⚠️</emoji> ᴀᴄᴛɪᴏɴ:</b>
<blockquote>– ᴍᴇssᴀɢᴇs ᴡɪᴛʜ sᴛʏʟɪsʜ ᴛᴇxᴛ ᴡɪʟʟ ʙᴇ ᴅᴇʟᴇᴛᴇᴅ ᴀғᴛᴇʀ 30 sᴇᴄᴏɴᴅs.
– ᴀ ᴡᴀʀɴɪɴɢ ᴡɪʟʟ ʙᴇ sᴇɴᴛ ᴀɴᴅ ᴀʟsᴏ ᴅᴇʟᴇᴛᴇᴅ ᴀғᴛᴇʀ 30 sᴇᴄᴏɴᴅs.</blockquote>

<b><emoji id='5420323339723881652'>ℹ️</emoji> ɴᴏᴛᴇ:</b>
<blockquote>– ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nostylish",
				Handler: RequirePermission("delete_messages", NoStylishCmd),
			},
			{
				Pattern: "callback:^nostylish_toggle:(on|off)$",
				Handler: handleNoStylishToggle,
			},
		},
	})
}

func getNoStylishDashboard(enabled bool) (string, telegram.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("🔡 <b>Stylish Font Filter</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n\n"+
		"<blockquote expandable>"+
		"When enabled, I scan for and remove messages sent with decorative or stylish fonts.\n\n"+
		"<b>Why?</b> These fonts are often used to bypass filters and can be difficult for screen readers and some users to read."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> Detected messages and their warnings are automatically deleted after 30 seconds.", status)

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data(toggleLabel, "nostylish_toggle:"+toggleVal),
		telegram.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard.Build()
}

func handleNoStylishToggle(c *telegram.CallbackQuery) error {
	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	if err := database.SetNoStylishEnabled(c.ChatID, enable); err != nil {
		c.Answer("⚠️ Update failed.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	text, keyboard := getNoStylishDashboard(enable)
	c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	c.Answer("✓ Settings Updated")
	return nil
}

func NoStylishCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if len(args) < 1 {
		isEnabled, err := database.IsNoStylishEnabled(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/nostylish on</code> — Block stylish fonts\n• <code>/nostylish off</code> — Allow stylish fonts",
			)
			return err
		}

		text, keyboard := getNoStylishDashboard(isEnabled)
		_, err = m.Reply(text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nostylish on</code> or <code>/nostylish off</code>.",
		)
		return err
	}

	if err := database.SetNoStylishEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	text, keyboard := getNoStylishDashboard(enable)
	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NoStylishHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() || m.IsChannelPost() || m.IsPrivate() {
		return nil
	}

	isEnabled, err := database.IsNoStylishEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	text := m.Text()
	if text == "" {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	if containsStylishFont(text) {
		mention := utils.MentionHTML(m.Sender)
		warnMsg, err := m.Reply(fmt.Sprintf(
			"⚠️ <b>Stylish Font Detected</b>\n\n"+
				"<i>%s, please use a standard normal font. Decorative or stylish text is not allowed here to maintain readability.</i>\n\n"+
				"💡 <b>Tip:</b> This message and yours will be automatically deleted in 30 seconds.",
			mention,
		), &telegram.SendOptions{ParseMode: "HTML"})

		// Schedule original message deletion
		ScheduleMessageDeletion(
			m.Client,
			m.ChatID(),
			int32(m.ID),
			30*time.Second,
		)

		// Schedule warning message deletion
		if err == nil && warnMsg != nil {
			ScheduleMessageDeletion(
				m.Client,
				m.ChatID(),
				int32(warnMsg.ID),
				30*time.Second,
			)
		}

		return telegram.ErrEndGroup
	}

	return nil
}

func containsStylishFont(text string) bool {
	for _, r := range text {
		if isStylish(r) {
			return true
		}
	}
	return false
}

func isStylish(r rune) bool {
	// Fast-path for standard text
	if r < 0x1D00 {
		return false
	}

	switch {
	case r >= 0x1D400 && r <= 0x1D7FF: // 𝓢𝗕𝘐 Mathematical
		return true
	case r >= 0x1D00 && r <= 0x1D7F: // ꜱᴍᴀʟʟ Small Caps
		return true
	case r >= 0xFF01 && r <= 0xFF5E: // Ａ Ｂ Ｃ Fullwidth
		return true
	case r >= 0x2460 && r <= 0x24FF: // ①② Enclosed
		return true
	default:
		return false
	}
}
