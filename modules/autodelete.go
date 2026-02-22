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
			_, err = m.Reply("⚠️ Usage:\n<code>/msgdelete on</code> — Enable\n<code>/msgdelete off</code> — Disable")
			return err
		} else {
			status := map[bool]string{true: "Enabled", false: "Disabled"}[settings.Enabled]
			_, err = m.Reply(fmt.Sprintf("📊 <b>MsgDelete Status:</b> %s\n⏱ <b>Delay:</b> %s\n🛡️ <b>Note:</b> Admins are always exempt", status, utils.FormatDuration(settings.Delay)))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply("❌ Invalid option.\nUse <code>/msgdelete on</code> or <code>/msgdelete off</code>")
		return err
	}

	if err := database.SetMsgDeleteEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply("❌ Failed to update setting.")
		return err
	}

	status := "💬 Message auto-delete enabled ✅"
	if !enable {
		status = "🚫 Message auto-delete disabled ❌"
	}
	_, err = m.Reply(status)
	return err
}

func SetMsgDelayCmd(m *telegram.NewMessage) error {
	var err error
	if _, err = m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err = m.Reply("⚠️ Usage: <code>/setmsgdelay 5m</code> or <code>/setmsgdelay 2h</code>\n\nValid range: 1m to 12h")
		return err
	}

	duration, err := utils.ParseDuration(m.Args())
	if err != nil {
		_, err = m.Reply("❌ Invalid time format. Use: 5m, 1h, 12h, etc.\nRange: 1 minute to 12 hours")
		return err
	}

	if duration < time.Minute || duration > 12*time.Hour {
		_, err = m.Reply("❌ Delay must be between 1 minute and 12 hours.")
		return err
	}

	if err := database.SetMsgDeleteDelay(m.ChatID(), duration); err != nil {
		_, err = m.Reply("❌ Failed to set delay.")
		return err
	}

	_, err = m.Reply(fmt.Sprintf("✅ Message auto-delete delay set to %s", utils.FormatDuration(duration)))
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


func updateReadChannelOutboxHandler(m telegram.Update, c *telegram.Client) error {

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

