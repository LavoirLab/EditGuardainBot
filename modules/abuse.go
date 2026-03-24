package modules

import (
	"fmt"
	"log"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

const (
	noAbuseInvalidMsg = `❌ <b>Invalid usage</b>

Please use one of the following formats:
• <code>/noabuse on</code> — To enable the filter
• <code>/noabuse off</code> — To disable the filter`

	noAbuseRemovedMsg = `🚨 <b>Abuse Detected!</b>

%s, <i>your message was removed because it contained abusive or offensive language. Please maintain a respectful environment.</i> 🛡️`

	noAbuseErrMsg = `❌ <b>Update failed</b>
<i>Please try again later.</i>`
)

func getNoAbuseDashboard(enabled bool) (string, telegram.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("🛡️ <b>NoAbuse Filter</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n\n"+
		"<blockquote expandable>"+
		"When enabled, I automatically detect and filter abusive or offensive language in group messages.\n\n"+
		"<b>Action:</b> Messages containing blacklisted words are deleted immediately to maintain a respectful environment."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> This feature is highly effective at keeping group chats clean and professional.", status)

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data(toggleLabel, "noabuse_toggle:"+toggleVal),
		telegram.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard.Build()
}

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "abuse",
		ModName: "🚫 Abuse",
		Help: `<b><emoji id='5929345782560853059'>🚫</emoji> Nᴏᴀʙᴜsᴇ ғɪʟᴛᴇʀ</b>

<i>ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇᴛᴇᴄᴛs ᴀɴᴅ ғɪʟᴛᴇʀs ᴀʙᴜsɪᴠᴇ ᴏʀ ᴏғғᴇɴsɪᴠᴇ ʟᴀɴɢᴜᴀɢᴇ ɪɴ ɢʀᴏᴜᴘ ᴍᴇssᴀɢᴇs.</i>

<b><emoji id='5341715473882955310'>🔧</emoji> ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• /noabuse on – ᴇɴᴀʙʟᴇ ᴀʙᴜsᴇ ᴅᴇᴛᴇᴄᴛɪᴏɴ <emoji id='6084551628461967966'>✅</emoji>
• /noabuse off – ᴅɪsᴀʙʟᴇ ᴀʙᴜsᴇ ᴅᴇᴛᴇᴄᴛɪᴏɴ <emoji id='5465665476971471368'>❌</emoji></blockquote>

<b><emoji id='5443038326535759644'>ℹ️</emoji> ɴᴏᴛᴇs:</b>
<blockquote>– ᴍᴇssᴀɢᴇs ᴡɪᴛʜ ᴏғғᴇɴsɪᴠᴇ ᴄᴏɴᴛᴇɴᴛ ᴡɪʟʟ ʙᴇ ᴄᴇɴsᴏʀᴇᴅ ᴏʀ ʀᴇᴍᴏᴠᴇᴅ.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>

<b><i><emoji id='5462921117423384478'>🔧</emoji> ᴛʜɪs ғᴇᴀᴛᴜʀᴇ ɪs ᴜɴᴅᴇʀ ᴅᴇᴠᴇʟᴏᴘᴍᴇɴᴛ ᴀɴᴅ ᴍᴀʏ ɴᴏᴛ ᴡᴏʀᴋ ᴘʀᴏᴘᴇʀʟʏ ᴀᴛ ᴛʜɪs ᴍᴏᴍᴇɴᴛ.</i></b>`,

		Handlers: []HandlerConfig{
			{
				Pattern: "command:noabuse",
				Handler: RequirePermission("delete_messages", NoAbuseCmd),
			},
			{
				Pattern: "callback:^noabuse_toggle:(on|off)$",
				Handler: handleNoAbuseToggle,
			},
		},
	})
}

func handleNoAbuseToggle(c *telegram.CallbackQuery) error {
	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	if err := database.SetNSFWFlag(c.ChatID, enable); err != nil {
		c.Answer("⚠️ Update failed.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	text, keyboard := getNoAbuseDashboard(enable)
	c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	c.Answer("✓ Settings Updated")
	return nil
}

func NoAbuseCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())

	if _, err = m.Delete(); err != nil {
		return err
	}

	// show status when no argument
	if len(args) < 1 {
		enabled := database.IsNSFWEnabled(m.ChatID())
		text, keyboard := getNoAbuseDashboard(enabled)
		_, err = m.Reply(text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err := m.Reply(noAbuseInvalidMsg)
		return err
	}

	if err := database.SetNSFWFlag(m.ChatID(), enable); err != nil {
		log.Println("NoAbuse.error:", err)
		_, err = m.Reply(noAbuseErrMsg)
		return err
	}

	text, keyboard := getNoAbuseDashboard(enable)
	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func AbuseHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	if !database.IsNSFWEnabled(m.ChatID()) {
		return nil
	}

	if isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID()); isadmin {
		return nil
	}

	isNsfw := utils.ContainsAbusiveWord(m.Text())
	if !isNsfw {
		return nil
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	user := utils.MentionHTML(m.Sender)
	_, _ = m.Reply(fmt.Sprintf(noAbuseRemovedMsg, user))
	return telegram.ErrEndGroup
}
