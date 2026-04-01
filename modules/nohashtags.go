package modules

import (
	"fmt"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func getNoHashtagsDashboard(enabled bool) (string, telegram.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("🏷️ <b>Hashtag Filter</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n\n"+
		"<blockquote expandable>"+
		"When enabled, I automatically detect and remove messages containing hashtags (#example).\n\n"+
		"<b>Action:</b> Hashtags are often used for spam or to bypass keyword filters. This protection ensures your chat remains focused and clutter-free."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> Use <code>/nohashtags on</code> or <code>/nohashtags off</code> to quickly toggle this setting.", status)

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data(toggleLabel, "nohashtags_toggle:"+toggleVal),
		telegram.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard
}

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nohashtags",
		ModName: "# Hashtags",
		Help: `# <b>ʜᴀsʜᴛᴀɢ ғɪʟᴛᴇʀ</b> ʙʟᴏᴄᴋs ᴍᴇssᴀɢᴇs ᴄᴏɴᴛᴀɪɴɪɴɢ ʜᴀsʜᴛᴀɢs.

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/nohashtags on</code> — ʙʟᴏᴄᴋ ʜᴀsʜᴛᴀɢs <emoji id='6084551628461967966'>✅</emoji>
• <code>/nohashtags off</code> — ᴀʟʟᴏᴡ ʜᴀsʜᴛᴀɢs <emoji id='5465665476971471368'>❌</emoji></blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nohashtags",
				Handler: RequirePermission("delete_messages", NoHashtagsCmd),
			},
			{
				Pattern: "command:nohashtag",
				Handler: RequirePermission("delete_messages", NoHashtagsCmd),
			},
			{
				Pattern: "callback:^nohashtags_toggle:(on|off)$",
				Handler: handleNoHashtagsToggle,
			},
		},
	})
}

func handleNoHashtagsToggle(c *telegram.CallbackQuery) error {
	if isAdmin, _ := utils.IsAdmin(c.Client, c.ChatID, c.SenderID); !isAdmin {
		c.Answer(
			"👤 Admin Access Required",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	if err := database.SetNoHashtagsEnabled(c.ChatID, enable); err != nil {
		c.Answer("⚠️ Update failed.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	text, keyboard := getNoHashtagsDashboard(enable)
	c.Answer("✓ Settings Updated")
	_, err := c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NoHashtagsCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, _ := database.IsNoHashtagsEnabled(m.ChatID())
		text, keyboard := getNoHashtagsDashboard(isEnabled)
		_, err = m.Reply(text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nohashtags on</code> or <code>/nohashtags off</code>.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	if err := database.SetNoHashtagsEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	text, keyboard := getNoHashtagsDashboard(enable)
	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NoHashtagsHandler(m *telegram.NewMessage) error {
	// Fast path: check for entities before any other processing
	if len(m.Message.Entities) == 0 || m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	// Optimization: Fast path check for hashtag entities before any expensive operations.
	// This avoids redundant database lookups and admin checks for messages without hashtags.
	hasHashtag := false
	for _, entity := range m.Message.Entities {
		if _, ok := entity.(*telegram.MessageEntityHashtag); ok {
			hasHashtag = true
			break
		}
	}

	if !hasHashtag {
		return nil
	}

	isEnabled, err := database.IsNoHashtagsEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	// Delete message with hashtags
	if _, err := m.Delete(); err != nil {
		return err
	}

	mention := utils.MentionHTML(m.Sender)
	_, _ = m.Reply(
		fmt.Sprintf(
			"🏷️ <b>No Hashtags!</b>\n\n%s, <i>hashtags are not permitted in this group to prevent spamming and maintain clarity.</i> 🚫",
			mention,
		),
	)
	return telegram.ErrEndGroup
}
