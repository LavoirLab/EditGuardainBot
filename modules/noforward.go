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
		Name:    "noforward",
		ModName: "📤 Forward",
		Help: `<emoji id='5433614747381538714'>📤</emoji> <b>ғᴏʀᴡᴀʀᴅ ᴄᴏɴᴛʀᴏʟ</b> ʙʟᴏᴄᴋs ғᴏʀᴡᴀʀᴅᴇᴅ ᴍᴇssᴀɢᴇs ɪɴ ᴛʜᴇ ɢʀᴏᴜᴘ.

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/noforward on</code> — ʙʟᴏᴄᴋ ғᴏʀᴡᴀʀᴅᴇᴅ ᴍᴇssᴀɢᴇs <emoji id='6084551628461967966'>✅</emoji>
• <code>/noforward off</code> — ᴀʟʟᴏᴡ ғᴏʀᴡᴀʀᴅᴇᴅ ᴍᴇssᴀɢᴇs <emoji id='5465665476971471368'>❌</emoji></blockquote>

<b><emoji id='5420323339723881652'>ℹ️</emoji> ɴᴏᴛᴇ:</b>
<blockquote>– ғᴏʀᴡᴀʀᴅᴇᴅ ᴍᴇssᴀɢᴇs ᴡɪʟʟ ʙᴇ ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇʟᴇᴛᴇᴅ.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:noforward",
				Handler: RequirePermission("delete_messages", NoForwardCmd),
			},
		},
	})
}

func NoForwardCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, err := database.IsNoForwardEnabled(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/noforward on</code> — Block forwarded messages\n• <code>/noforward off</code> — Allow forwarded messages",
			)
			return err
		} else {
			status := "Disabled ❌"
			if isEnabled {
				status = "Enabled ✅"
			}
			_, err = m.Reply(fmt.Sprintf("📤 <b>Forward Control Status:</b> %s\n\n<i>When enabled, forwarded messages from other chats will be automatically removed.</i>", status))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/noforward on</code> or <code>/noforward off</code>.",
		)
		return err
	}

	if err := database.SetNoForwardEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/noforward off</code> to allow forwarded messages again."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/noforward on</code> to block forwarded messages."
	}
	_, err = m.Reply(
		fmt.Sprintf("📤 <b>Forward control %s</b>\n\n%s", status, next),
	)
	return err
}

func NoForwardHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
		return nil
	}

	isEnabled, err := database.IsNoForwardEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	// Check if message is forwarded
	if m.Message.FwdFrom == nil {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	// Delete forwarded message
	if _, err := m.Delete(); err != nil {
		return err
	}

	mention := utils.MentionHTML(m.Sender)
	mm, _ := m.Reply(
		fmt.Sprintf(
			"📤 <b>NoForward Enabled!</b>\n\n%s, <i>forwarding messages is restricted in this group to maintain original conversations.</i> 🚫",
			mention,
		),
	)
	if mm != nil {
		ScheduleDeletion(mm, 30*time.Second)
	}
	return telegram.ErrEndGroup
}
