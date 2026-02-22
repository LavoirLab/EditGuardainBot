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
	noAbuseStatusEnabled = `🛡️ <b>NoAbuse</b>
<b>Status:</b> <i>Enabled</i>

<i>Abusive messages are automatically removed.</i>

<i>Use <code>/noabuse off</code> to turn it off.</i>`

	noAbuseStatusDisabled = `🚫 <b>NoAbuse</b>
<b>Status:</b> <i>Disabled</i>

<i>Messages are not filtered.</i>

<i>Use <code>/noabuse on</code> to turn it on.</i>`

	noAbuseEnabledMsg = `🛡️ <b>NoAbuse Enabled</b>

<i>Abusive messages will now be deleted automatically. Use <code>/noabuse off</code> to disable this filter.</i>`

	noAbuseDisabledMsg = `🚫 <b>NoAbuse Disabled</b>

<i>Messages will no longer be filtered for abusive language. Use <code>/noabuse on</code> to enable it again.</i>`

	noAbuseInvalidMsg = `❌ <b>Invalid usage</b>

Please use one of the following formats:
• <code>/noabuse on</code> — To enable the filter
• <code>/noabuse off</code> — To disable the filter`

	noAbuseRemovedMsg = `🚨 <b>Abuse Detected!</b>

%s, <i>your message was removed because it contained abusive or offensive language. Please maintain a respectful environment.</i> 🛡️`

	noAbuseErrMsg = `❌ <b>Update failed</b>
<i>Please try again later.</i>`
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "abuse",
		ModName: "🚫 Abuse",
		Help: `<b><emoji id='5929345782560853059'>🚫</emoji> Nᴏᴀʙᴜsᴇ ғɪʟᴛᴇʀ</b>

<i>ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇᴛᴇᴄᴛs ᴀɴᴅ ғɪʟᴛᴇʀs ᴀʙᴜsɪᴠᴇ ᴏʀ ᴏғғᴇɴsɪᴠᴇ ʟᴀɴɢᴜᴀɢᴇ ɪɴ ɢʀᴏᴜᴘ ᴍᴇssᴀɢᴇs.</i>

<b><emoji id='5341715473882955310'>🔧</emoji> ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/noabuse on</code> – ᴇɴᴀʙʟᴇ ᴀʙᴜsᴇ ᴅᴇᴛᴇᴄᴛɪᴏɴ <emoji id='6084551628461967966'>✅</emoji>
• <code>/noabuse off</code> – ᴅɪsᴀʙʟᴇ ᴀʙᴜsᴇ ᴅᴇᴛᴇᴄᴛɪᴏɴ <emoji id='5465665476971471368'>❌</emoji></blockquote>

<b><emoji id='5443038326535759644'>ℹ️</emoji> ɴᴏᴛᴇs:</b>
<blockquote>– ᴍᴇssᴀɢᴇs ᴡɪᴛʜ ᴏғғᴇɴsɪᴠᴇ ᴄᴏɴᴛᴇɴᴛ ᴡɪʟʟ ʙᴇ ᴄᴇɴsᴏʀᴇᴅ ᴏʀ ʀᴇᴍᴏᴠᴇᴅ.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>

<blockquote><emoji id='5462921117423384478'>🔧</emoji> ᴛʜɪs ғᴇᴀᴛᴜʀᴇ ɪs ᴜɴᴅᴇʀ ᴅᴇᴠᴇʟᴏᴘᴍᴇɴᴛ ᴀɴᴅ ᴍᴀʏ ɴᴏᴛ ᴡᴏʀᴋ ᴘʀᴏᴘᴇʀʟʏ ᴀᴛ ᴛʜɪs ᴍᴏᴍᴇɴᴛ.</blockquote>`,

		Handlers: []HandlerConfig{
			{
				Pattern: "command:noabuse",
				Handler: RequirePermission("delete_messages", NoAbuseCmd),
			},
		},
	})
}

func NoAbuseCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())

	if _, err = m.Delete(); err != nil {
		return err
	}

	// show status when no argument
	if len(args) < 1 {
		if database.IsNSFWEnabled(m.ChatID()) {
			_, err = m.Reply(noAbuseStatusEnabled)
			return err
		}
		_, err = m.Reply(noAbuseStatusDisabled)
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

	if enable {
		_, err = m.Reply(noAbuseEnabledMsg)
		return err
	}

	_, err = m.Reply(noAbuseDisabledMsg)
	return err
}

func AbuseHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
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
