package modules

import (
	"fmt"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

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
		},
	})
}

func NoHashtagsCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, err := database.IsNoHashtagsEnabled(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/nohashtags on</code> — Block hashtags\n• <code>/nohashtags off</code> — Allow hashtags",
			)
			return err
		} else {
			status := "Disabled ❌"
			if isEnabled {
				status = "Enabled ✅"
			}
			_, err = m.Reply(fmt.Sprintf("🏷️ <b>Hashtag Filter Status:</b> %s\n\n<i>When enabled, messages containing hashtags (#example) will be automatically removed.</i>", status))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nohashtags on</code> or <code>/nohashtags off</code>.",
		)
		return err
	}

	if err := database.SetNoHashtagsEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/nohashtags off</code> to allow hashtags again."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/nohashtags on</code> to block hashtags."
	}
	_, err = m.Reply(
		fmt.Sprintf("🏷️ <b>Hashtag filter %s</b>\n\n%s", status, next),
	)
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
