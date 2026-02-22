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
		Name:    "mediadelete",
		ModName: "🎬 Media",
		Help: `<emoji id='5368653135101310687'>🎬</emoji> <b>Mᴇᴅɪᴀ ᴀᴜᴛᴏ-ᴅᴇʟᴇᴛᴇ</b> ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ʀᴇᴍᴏᴠᴇs ᴍᴇᴅɪᴀ ᴍᴇssᴀɢᴇs ᴀғᴛᴇʀ ᴀ sᴇᴛ ᴅᴇʟᴀʏ.

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/mediadelete on</code> — ᴇɴᴀʙʟᴇ ᴍᴇᴅɪᴀ ᴀᴜᴛᴏ-ᴅᴇʟᴇᴛᴇ <emoji id='6084551628461967966'>✅</emoji>
• <code>/mediadelete off</code> — ᴅɪsᴀʙʟᴇ ᴍᴇᴅɪᴀ ᴀᴜᴛᴏ-ᴅᴇʟᴇᴛᴇ <emoji id='5465665476971471368'>❌</emoji>
• <code>/setmediadelay &lt;time&gt;</code> — sᴇᴛ ᴅᴇʟᴀʏ (1m ᴛᴏ 24h)
</blockquote>

<b>ᴛɪᴍᴇ ғᴏʀᴍᴀᴛ:</b>
• <code>5m</code> = 5 ᴍɪɴᴜᴛᴇs
• <code>1h</code> = 1 ʜᴏᴜʀ
• <code>12h</code> = 12 ʜᴏᴜʀs

<blockquote><b><emoji id='5420323339723881652'>ℹ️</emoji> ɴᴏᴛᴇs:</b>
– ᴀᴘᴘʟɪᴇs ᴛᴏ ᴘʜᴏᴛᴏs, ᴠɪᴅᴇᴏs, sᴛɪᴄᴋᴇʀs, ɢɪғs, ʟᴏᴄᴀᴛɪᴏɴ, ᴘᴏʟʟ ᴀɴᴅ ᴀɴɪᴍᴀᴛɪᴏɴs.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:mediadelete",
				Handler: RequirePermission("change_info", MediaDeleteCmd),
			},
			{
				Pattern: "command:setmediadelay",
				Handler: RequirePermission("change_info", SetMediaDelayCmd),
			},
		},
	})
}

func MediaDeleteCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		settings, err := database.GetMediaDeleteSettings(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/mediadelete on</code> — Enable media auto-delete\n• <code>/mediadelete off</code> — Disable media auto-delete",
			)
			return err
		} else {
			status := "Disabled ❌"
			if settings.Enabled {
				status = "Enabled ✅"
			}

			_, err = m.Reply(fmt.Sprintf(
				"🎬 <b>Media Auto-Delete Status:</b> %s\n⏱ <b>Delay:</b> %s\n\n<i>When enabled, all media messages will be automatically removed after the set delay.</i>",
				status,
				utils.FormatDuration(settings.Delay),
			))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/mediadelete on</code> or <code>/mediadelete off</code>.",
		)
		return err
	}

	if err := database.SetMediaDeleteEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/mediadelete off</code> to allow media again."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/mediadelete on</code> to block media."
	}
	_, err = m.Reply(
		fmt.Sprintf("🎬 <b>Media auto-delete %s</b>\n\n%s", status, next),
	)
	return err
}

func SetMediaDelayCmd(m *telegram.NewMessage) error {
	var err error
	if _, err = m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err = m.Reply(
			"⚠️ <b>Usage:</b> <code>/setmediadelay &lt;time&gt;</code>\n\n<b>Examples:</b>\n• <code>/setmediadelay 5m</code>\n• <code>/setmediadelay 2h</code>\n\n<i>Valid range: 1 minute to 24 hours.</i>",
		)
		return err
	}

	duration, err := utils.ParseDuration(m.Args())
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid time format.</b>\n\nPlease use formats like: <code>5m</code>, <code>1h</code>, <code>12h</code>.\n<i>Range: 1 minute to 24 hours.</i>",
		)
		return err
	}

	if duration < time.Minute || duration > 24*time.Hour {
		_, err = m.Reply(
			"❌ <b>Out of range.</b>\n\nDelay must be between 1 minute and 24 hours.",
		)
		return err
	}

	if err := database.SetMediaDeleteDelay(m.ChatID(), duration); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Failed to set delay.</b>\n\nSomething went wrong while saving the delay. Please try again.",
		)
		return err
	}

	_, err = m.Reply(
		fmt.Sprintf(
			"✅ <b>Media auto-delete delay set to %s!</b>\n\nMedia will now be removed after this duration.",
			utils.FormatDuration(duration),
		),
	)
	return err
}

func MediaDeleteHandler(m *telegram.NewMessage) error {
	if !m.IsMedia() || m.IsAnonymous() {
		return nil
	}

	settings, err := database.GetMediaDeleteSettings(m.ChatID())
	if err != nil || !settings.Enabled {
		return nil
	}

	ScheduleMessageDeletion(m.Client, m.ChatID(), int32(m.ID), settings.Delay)

	return nil
}
