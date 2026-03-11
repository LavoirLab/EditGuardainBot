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
		},
	})
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
		} else {
			status := "Disabled 🔴"
			if isEnabled {
				status = "Enabled 🟢"
			}
			_, err = m.Reply(fmt.Sprintf("🔡 <b>Stylish Font Filter Status:</b> %s\n\n<i>When enabled, messages with decorative or stylish fonts will be removed to maintain readability.</i>\n\n<i>💡 Tip: Use <code>/nostylish on</code> to toggle.</i>", status))
			return err
		}
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

	status := "enabled 🟢"
	next := "Use <code>/nostylish off</code> to allow stylish fonts again."
	if !enable {
		status = "disabled 🔴"
		next = "Use <code>/nostylish on</code> to block stylish fonts."
	}
	_, err = m.Reply(
		fmt.Sprintf("🔡 <b>Stylish Font Filter %s</b>\n\n%s", status, next),
	)
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
			"⚠️ %s, Please use a standard normal font. Stylish text is not allowed here and this message will be deleted in 30 seconds.",
			mention,
		))

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
