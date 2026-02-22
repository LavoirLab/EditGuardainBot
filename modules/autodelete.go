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
		Help: "💬 <b>Message Auto-Delete</b> automatically removes all messages after a set delay.\n\n" +
			"<b>Commands:</b>\n" +
			"• <code>/msgdelete on</code> — Enable message auto-delete ✅\n" +
			"• <code>/msgdelete off</code> — Disable message auto-delete ❌\n" +
			"• <code>/setmsgdelay &lt;time&gt;</code> — Set delay (1m to 12h)\n\n" +
			"<b>Time Format:</b>\n" +
			"• <code>5m</code> = 5 minutes\n" +
			"• <code>1h</code> = 1 hour\n" +
			"• <code>12h</code> = 12 hours\n\n" +
			"<b>ℹ️ Note:</b> Applies to all text and media messages from regular users only.\n" +
			"👮 Only group admins can configure this setting.",
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
