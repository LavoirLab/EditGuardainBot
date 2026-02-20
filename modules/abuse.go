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

<i>Abusive messages will now be deleted.</i>`

	noAbuseDisabledMsg = `🚫 <b>NoAbuse Disabled</b>

<i>Messages will no longer be filtered.</i>`

	noAbuseInvalidMsg = `❌ <b>Invalid usage</b>

<code>/noabuse on</code>
<code>/noabuse off</code>`

	noAbuseRemovedMsg = `🚫 <b>Message Removed</b>

%s, <i>abusive language is not allowed here.</i>`

	noAbuseErrMsg = `❌ <b>Update failed</b>
<i>Please try again later.</i>`
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "abuse",
		ModName: "🚫 Abuse",
		Help: "<b>🚫 NoAbuse Filter</b>\n" +
			"Automatically detects and filters abusive or offensive language in group messages.\n\n" +
			"<b>🔧 Commands:</b>\n" +
			"• <code>/noabuse on</code> – Enable abuse detection ✅\n" +
			"• <code>/noabuse off</code> – Disable abuse detection ❌\n\n" +
			"<b>ℹ️ Notes:</b>\n" +
			"– Messages with offensive content will be censored or removed.\n" +
			"– 👮 Only group admins can configure this setting.\n\n" +
			"<blockquote>🔧 This feature is under development and may not work properly at this moment.</blockquote>",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:noabuse",
				Handler: RequirePermission("delete_messages", NoAbuseCmd),
			},
			{
				Pattern: "message",
				Handler: DeleteAbuseHandle,
			},
		},
	})
}

func NoAbuseCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Args())

	if _, err := m.Delete(); err != nil {
		return err
	}

	// show status when no argument
	if len(args) < 1 {
		if database.IsNSFWEnabled(m.ChatID()) {
			_, err := m.Reply(noAbuseStatusEnabled)
			return err
		}
		_, err := m.Reply(noAbuseStatusDisabled)
		return err
	}

	arg := strings.ToLower(args[0])

	if arg != "on" && arg != "off" {
		_, err := m.Reply(noAbuseInvalidMsg)
		return err
	}

	enable := arg == "on"

	if err := database.SetNSFWFlag(m.ChatID(), enable); err != nil {
		log.Println("NoAbuse.error:", err)
		_, err := m.Reply(noAbuseErrMsg)
		return err
	}

	if enable {
		_, err := m.Reply(noAbuseEnabledMsg)
		return err
	}

	_, err := m.Reply(noAbuseDisabledMsg)
	return err
}

func DeleteAbuseHandle(m *telegram.NewMessage) error {
	if !m.IsGroup() || m.Channel == nil || m.IsAnonymous() || m.IsChannelPost() {
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
	_, err := m.Reply(fmt.Sprintf(noAbuseRemovedMsg, user))
	return err
}