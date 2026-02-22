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
		Name:    "autodelete",
		ModName: "💬 MsgDelete",
		Help: `<emoji id='5443038326535759644'>💬</emoji> <b>Mᴇssᴀɢᴇ ᴀᴜᴛᴏ-ᴅᴇʟᴇᴛᴇ</b> 

<i>ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ʀᴇᴍᴏᴠᴇs ᴀʟʟ ɢʀᴏᴜᴘ ᴍᴇssᴀɢᴇs ᴀғᴛᴇʀ ᴀ ᴄᴏɴғɪɢᴜʀᴇᴅ ᴅᴇʟᴀʏ, ᴋᴇᴇᴘɪɴɢ ᴄʜᴀᴛs ᴄʟᴇᴀɴ, ʟɪɢʜᴛᴡᴇɪɢʜᴛ ᴀɴᴅ ғʀᴇᴇ ғʀᴏᴍ ᴍᴇssᴀɢᴇ ᴄʟᴜᴛᴛᴇʀ.</i>

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/msgdelete on</code> — <b>ᴇɴᴀʙʟᴇ ᴍᴇssᴀɢᴇ ᴀᴜᴛᴏ-ᴅᴇʟᴇᴛᴇ <emoji id='6084551628461967966'>✅</emoji></b>
• <code>/msgdelete off</code> — <b>ᴅɪsᴀʙʟᴇ ᴍᴇssᴀɢᴇ ᴀᴜᴛᴏ-ᴅᴇʟᴇᴛᴇ <emoji id='5465665476971471368'>❌</emoji></b>
• <code>/setmsgdelay &lt;time&gt;</code> — <b>sᴇᴛ ᴅᴇʟᴀʏ (1m ᴛᴏ 12h)</b></blockquote>

<blockquote><b>ᴛɪᴍᴇ ғᴏʀᴍᴀᴛ:</b>
• <code>5m</code> = 5 ᴍɪɴᴜᴛᴇs
• <code>1h</code> = 1 ʜᴏᴜʀ
• <code>12h</code> = 12 ʜᴏᴜʀs</blockquote>

<b><emoji id='5420323339723881652'>ℹ️</emoji> ɴᴏᴛᴇs:</b><blockquote>
– ᴀᴘᴘʟɪᴇs ᴛᴏ ᴀʟʟ ᴛᴇxᴛ ᴀɴᴅ ᴍᴇᴅɪᴀ ᴍᴇssᴀɢᴇs ғʀᴏᴍ ʀᴇɢᴜʟᴀʀ ᴜsᴇʀs ᴏɴʟʏ.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:msgdelete",
				Handler: RequirePermission("change_info", MsgDeleteCmd),
			},
			{
				Pattern: "command:setmsgdelay",
				Handler: RequirePermission("change_info", SetMsgDelayCmd),
			},
			{
				Pattern: "*",
				Handler: updateReadChannelOutboxHandler,
			},
		},
	})
}

func MsgDeleteCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		settings, err := database.GetMsgDeleteSettings(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/msgdelete on</code> — Enable auto-delete\n• <code>/msgdelete off</code> — Disable auto-delete",
			)
			return err
		} else {
			status := "Disabled ❌"
			if settings.Enabled {
				status = "Enabled ✅"
			}
			_, err = m.Reply(fmt.Sprintf("💬 <b>Message Auto-Delete Status:</b> %s\n⏱ <b>Delay:</b> %s\n\n<i>When enabled, messages will be automatically removed after the set delay. Admins are exempt.</i>", status, utils.FormatDuration(settings.Delay)))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/msgdelete on</code> or <code>/msgdelete off</code>.",
		)
		return err
	}

	if err := database.SetMsgDeleteEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/msgdelete off</code> to disable it."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/msgdelete on</code> to enable it again."
	}
	_, err = m.Reply(
		fmt.Sprintf("💬 <b>Message auto-delete %s</b>\n\n%s", status, next),
	)
	return err
}

func SetMsgDelayCmd(m *telegram.NewMessage) error {
	var err error
	if _, err = m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err = m.Reply(
			"⚠️ <b>Usage:</b> <code>/setmsgdelay &lt;time&gt;</code>\n\n<b>Examples:</b>\n• <code>/setmsgdelay 5m</code>\n• <code>/setmsgdelay 1h</code>\n\n<i>Valid range: 1 minute to 12 hours.</i>",
		)
		return err
	}

	duration, err := utils.ParseDuration(m.Args())
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid time format.</b>\n\nPlease use formats like: <code>5m</code>, <code>1h</code>, <code>12h</code>.\n<i>Range: 1 minute to 12 hours.</i>",
		)
		return err
	}

	if duration < time.Minute || duration > 12*time.Hour {
		_, err = m.Reply(
			"❌ <b>Out of range.</b>\n\nDelay must be between 1 minute and 12 hours.",
		)
		return err
	}

	if err := database.SetMsgDeleteDelay(m.ChatID(), duration); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Failed to set delay.</b>\n\nSomething went wrong while saving the delay. Please try again.",
		)
		return err
	}

	_, err = m.Reply(
		fmt.Sprintf(
			"✅ <b>Message auto-delete delay set to %s!</b>\n\nMessages will now be removed after this duration.",
			utils.FormatDuration(duration),
		),
	)
	return err
}

func AutoDeleteHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
		return nil
	}
	settings, err := database.GetMsgDeleteSettings(m.ChatID())
	if err != nil || !settings.Enabled {
		return nil
	}

	ScheduleMessageDeletion(m.Client, m.ChatID(), int32(m.ID), settings.Delay)

	return nil
}

func updateReadChannelOutboxHandler(
	m telegram.Update,
	c *telegram.Client,
) error {
	upd, ok := m.(*telegram.UpdateReadChannelOutbox)
	if !ok {
		return nil
	}

	settings, err := database.GetMsgDeleteSettings(upd.ChannelID)
	if err != nil || !settings.Enabled {
		return nil
	}

	ScheduleMessageDeletion(c, upd.ChannelID, int32(upd.MaxID), settings.Delay)

	return nil
}
