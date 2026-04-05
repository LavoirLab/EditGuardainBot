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

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nobotpromo",
		ModName: "🤖 Bot Promo",
		Help: `<b>🤖 ɴᴏ ʙᴏᴛ ᴘʀᴏᴍᴏ</b>

<i>ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇᴛᴇᴄᴛs ᴀɴᴅ ʀᴇᴍᴏᴠᴇs ᴍᴇssᴀɢᴇs ᴘʀᴏᴍᴏᴛɪɴɢ ᴏᴛʜᴇʀ ʙᴏᴛs.</i>

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/nobotpromo on</code> — ᴇɴᴀʙʟᴇ ʙᴏᴛ ᴘʀᴏᴍᴏ ʙʟᴏᴄᴋɪɴɢ <emoji id='6084551628461967966'>✅</emoji>
• <code>/nobotpromo off</code> — ᴅɪsᴀʙʟᴇ ʙᴏᴛ ᴘʀᴏᴍᴏ ʙʟᴏᴄᴋɪɴɢ <emoji id='5465665476971471368'>❌</emoji></blockquote>

<b><emoji id='5420323339723881652'>ℹ️</emoji> ɴᴏᴛᴇ:</b>
<blockquote>– ᴍᴇssᴀɢᴇs ᴄᴏɴᴛᴀɪɴɪɴɢ ʙᴏᴛ ᴜsᴇʀɴᴀᴍᴇs ᴡɪʟʟ ʙᴇ ᴅᴇʟᴇᴛᴇᴅ ɪɴsᴛᴀɴᴛʟʏ.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nobotpromo",
				Handler: RequirePermission("delete_messages", NoBotPromoCmd),
			},
			{
				Pattern: "callback:^nobotpromo_toggle:(on|off)$",
				Handler: handleNoBotPromoToggle,
			},
		},
	})
}

func getNoBotPromoDashboard(enabled bool) (string, telegram.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("🤖 <b>No Bot Promo</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n\n"+
		"<blockquote expandable>"+
		"When enabled, I scan for messages containing bot usernames (e.g., @examplebot).\n\n"+
		"<b>Action:</b> Such messages are deleted immediately to prevent unauthorized bot promotions."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> This helps keep your group focused and free from bot spam.", status)

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data(toggleLabel, "nobotpromo_toggle:"+toggleVal),
		telegram.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard
}

func handleNoBotPromoToggle(c *telegram.CallbackQuery) error {
	if isAdmin, _ := utils.IsAdmin(c.Client, c.ChatID, c.SenderID); !isAdmin {
		c.Answer(
			"👤 Admin Access Required",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	if err := database.SetNoBotPromoEnabled(c.ChatID, enable); err != nil {
		c.Answer("⚠️ Update failed.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	text, keyboard := getNoBotPromoDashboard(enable)
	c.Answer("✓ Settings Updated")
	_, err := c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NoBotPromoCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, _ := database.IsNoBotPromoEnabled(m.ChatID())
		text, keyboard := getNoBotPromoDashboard(isEnabled)
		_, err = m.Reply(text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nobotpromo on</code> or <code>/nobotpromo off</code>.",
		)
		return err
	}

	if err := database.SetNoBotPromoEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	text, keyboard := getNoBotPromoDashboard(enable)
	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NoBotPromoHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	isEnabled, err := database.IsNoBotPromoEnabled(m.ChatID())
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

	if containsBotUsername(m.Text()) {
		if _, err := m.Delete(); err != nil {
			return err
		}

		mention := utils.MentionHTML(m.Sender)
		mm, _ := m.Reply(fmt.Sprintf(
			"🤖 <b>No Bot Promo!</b>\n\n%s, <i>promoting other bots is not allowed in this group.</i> 🚫",
			mention,
		))
		if mm != nil {
			ScheduleDeletion(mm, 30*time.Second)
		}

		return telegram.ErrEndGroup
	}

	return nil
}

func containsBotUsername(text string) bool {
	n := len(text)
	for i := 0; i < n; {
		for i < n && unicode.IsSpace(rune(text[i])) {
			i++
		}
		if i >= n {
			break
		}

		if text[i] == '@' {
			start := i
			for i < n && !unicode.IsSpace(rune(text[i])) {
				i++
			}
			word := text[start:i]
			if len(word) >= 4 {
				l := len(word)
				if (word[l-3] == 'b' || word[l-3] == 'B') &&
					(word[l-2] == 'o' || word[l-2] == 'O') &&
					(word[l-1] == 't' || word[l-1] == 'T') {
					return true
				}
			}
		} else {
			for i < n && !unicode.IsSpace(rune(text[i])) {
				i++
			}
		}
	}
	return false
}
