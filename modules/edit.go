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
                Name:    "edit",
                ModName: "✏️ Edit",
                Help: `<b><emoji id='5445214044325684784'>✏️</emoji> Eᴅɪᴛ Pʀᴏᴛᴇᴄᴛɪᴏɴ</b>

<i>ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇᴛᴇᴄᴛs ᴀɴᴅ ᴅᴇʟᴇᴛᴇs ᴍᴇssᴀɢᴇs ᴛʜᴀᴛ ʜᴀᴠᴇ ʙᴇᴇɴ ᴇᴅɪᴛᴇᴅ. ᴛʜɪs ᴇɴsᴜʀᴇs ᴛʜᴀᴛ ᴛʜᴇ ᴄᴏɴᴠᴇʀsᴀᴛɪᴏɴ ʀᴇᴍᴀɪɴs ᴛʀᴀɴsᴘᴀʀᴇɴᴛ ᴀɴᴅ ᴜɴᴀʟᴛᴇʀᴇᴅ.</i>

<b><emoji id='5341715473882955310'>🔧</emoji> ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• /edit on – ᴇɴᴀʙʟᴇ ᴇᴅɪᴛ ᴘʀᴏᴛᴇᴄᴛɪᴏɴ <emoji id='6084551628461967966'>✅</emoji>
• /edit off – ᴅɪsᴀʙʟᴇ ᴇᴅɪᴛ ᴘʀᴏᴛᴇᴄᴛɪᴏɴ <emoji id='5465665476971471368'>❌</emoji>
• /seteditdelay &lt;time&gt; – sᴇᴛ ᴅᴇʟᴀʏ ʙᴇғᴏʀᴇ ᴅᴇʟᴇᴛɪɴɢ ᴇᴅɪᴛᴇᴅ ᴍᴇssᴀɢᴇs (ᴇ.ɢ. 5s, 1m)</blockquote>

<b><emoji id='5443038326535759644'>ℹ️</emoji> ɴᴏᴛᴇs:</b>
<blockquote>– ᴇᴅɪᴛᴇᴅ ᴍᴇssᴀɢᴇs ᴡɪʟʟ ʙᴇ ᴅᴇʟᴇᴛᴇᴅ ɪᴍᴍᴇᴅɪᴀᴛᴇʟʏ ᴏʀ ᴀғᴛᴇʀ ᴛʜᴇ sᴇᴛ ᴅᴇʟᴀʏ.
– ɪғ ᴛʜᴇ ᴅᴇʟᴀʏ ɪs sᴇᴛ ᴛᴏ <b>0</b>, ᴛʜᴇ ᴍᴇssᴀɢᴇ ᴡɪʟʟ ʙᴇ ᴅᴇʟᴇᴛᴇᴅ ɪɴsᴛᴀɴᴛʟʏ.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>`,
                Handlers: []HandlerConfig{
                        {
                                Pattern: "edit",
                                Handler: deleteEditedMessage,
                        },
                        {
                                Pattern: "command:edit",
                                Handler: RequirePermission(
                                        "delete_messages",
                                        handleEditCommand,
                                ),
                        },
                        {
                                Pattern: "command:seteditdelay",
                                Handler: RequirePermission(
                                        "delete_messages",
                                        handleSetEditDelayCommand,
                                ),
                        },
                },
        })
}


func handleEditCommand(m *telegram.NewMessage) error {
	args := strings.Fields(m.Args())

	if len(args) < 1 {
		settings := database.GetEditMode(m.ChatID())
		status := "Enabled 🟢"
		if !settings.Enabled {
			status = "Disabled 🔴"
		}
		_, err := m.Reply(
			fmt.Sprintf(
				"✏️ <b>Edit Protection Status:</b> %s\n⏱ <b>Delay:</b> %s\n\n<i>Use <code>/edit on</code> or <code>/edit off</code> to change the setting.</i>",
				status,
				utils.FormatDuration(settings.Duration),
			),
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err := m.Reply(
			"❌ <b>Invalid argument!</b> Use <code>on</code> or <code>off</code>.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	err = database.SetEditMode(m.ChatID(), enable)
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Failed to update settings.</b>",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	statusMsg := "<b>Enabled</b> 🟢"
	if !enable {
		statusMsg = "<b>Disabled</b> 🔴"
	}

	_, err = m.Reply(
		fmt.Sprintf("✏️ <b>Edit Protection has been %s!</b>", statusMsg),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	return err
}

func handleSetEditDelayCommand(m *telegram.NewMessage) error {
	args := strings.Fields(m.Args())
	if len(args) < 1 {
		_, err := m.Reply(
			"⚠️ <b>Usage:</b> <code>/seteditdelay &lt;time&gt;</code>\n\n<b>Example:</b> <code>/seteditdelay 5s</code>, <code>1m</code>",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	duration, err := time.ParseDuration(args[0])
	if err != nil {
		// Fallback to utils.ParseDuration if standard ParseDuration fails
		duration, err = utils.ParseDuration(args[0])
		if err != nil {
			_, err = m.Reply(
				"❌ <b>Invalid time format.</b> Use formats like <code>5s</code>, <code>1m</code>, <code>1h</code>.",
				&telegram.SendOptions{ParseMode: "HTML"},
			)
			return err
		}
	}

	if duration < 0 {
		_, err = m.Reply(
			"❌ <b>Duration cannot be negative.</b>",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	err = database.SetEditDelay(m.ChatID(), duration)
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Failed to update delay.</b>",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	_, err = m.Reply(
		fmt.Sprintf(
			"✅ <b>Edit Protection delay set to %s!</b>",
			utils.FormatDuration(duration),
		),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	return err
}

func deleteEditedMessage(m *telegram.NewMessage) error {
	if !m.IsGroup() || m.Channel == nil || m.Message.EditHide ||
		m.IsAnonymous() ||
		m.IsChannelPost() {
		return nil
	}

	settings := database.GetEditMode(m.ChatID())
	if !settings.Enabled {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	if settings.Duration > 0 {
		ScheduleMessageDeletion(
			m.Client,
			m.ChatID(),
			int32(m.ID),
			settings.Duration,
		)
		return nil
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	senderTag := utils.MentionHTML(m.Sender)

	reason := fmt.Sprintf(
		`<b>🚫 %s edited a message.</b> Editing messages is prohibited in this chat to maintain conversation integrity.`,
		senderTag,
	)
	p := fmt.Sprintf(
		`<b>📷 %s edited a photo caption.</b> Image edits are blocked to preserve context.`,
		senderTag,
	)
	v := fmt.Sprintf(
		`<b>🎥 %s edited a video caption.</b> Video edits aren't allowed to retain originality.`,
		senderTag,
	)
	d := fmt.Sprintf(
		`<b>📄 %s edited a document caption.</b> Please avoid modifying documents.`,
		senderTag,
	)
	a := fmt.Sprintf(
		`<b>🎵 %s edited an audio caption.</b> Audio files must remain unaltered.`,
		senderTag,
	)

	switch {
	case m.Text() != "" && m.IsMedia():
		reason = fmt.Sprintf(
			`<b>✍️ %s edited a media caption.</b> Caption edits affect clarity and are not permitted.`,
			senderTag,
		)
		if m.Photo() != nil {
			reason = p
		} else if m.Video() != nil {
			reason = v
		} else if m.Document() != nil {
			reason = d
		} else if m.Audio() != nil {
			reason = a
		}
	case m.Text() != "" && !m.IsMedia():
		reason = fmt.Sprintf(
			`<b>🚫 %s edited text.</b> Editing text is not allowed to keep conversations clear.`,
			senderTag,
		)
	case m.Photo() != nil:
		reason = fmt.Sprintf(
			`<b>📷 %s edited a photo.</b> Editing photos is not allowed.`,
			senderTag,
		)
	case m.Video() != nil:
		reason = fmt.Sprintf(
			`<b>🎥 %s edited a video.</b> Editing videos is not permitted.`,
			senderTag,
		)
	case m.Document() != nil:
		reason = fmt.Sprintf(
			`<b>📄 %s edited a document.</b> Document edits are not allowed.`,
			senderTag,
		)
	case m.Audio() != nil:
		reason = fmt.Sprintf(
			`<b>🎵 %s edited an audio file.</b> Audio editing is not permitted.`,
			senderTag,
		)
	case m.Voice() != nil:
		reason = fmt.Sprintf(
			`<b>🎙️ %s edited a voice message.</b> Voice messages should remain original.`,
			senderTag,
		)
	case m.Animation() != nil:
		reason = fmt.Sprintf(
			`<b>🎞️ %s edited a GIF or animation.</b> Keep animations unchanged for context.`,
			senderTag,
		)
	}

	_, err := m.Reply(reason, &telegram.SendOptions{ParseMode: "HTML"})
	return err
}
