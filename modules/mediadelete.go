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
		ModName: "🎬 MediaDelete",
		Help: "🎬 <b>Media Auto-Delete</b> automatically removes media messages after a set delay.\n\n" +
			"<b>Commands:</b>\n" +
			"• <code>/mediadelete on</code> — Enable media auto-delete ✅\n" +
			"• <code>/mediadelete off</code> — Disable media auto-delete ❌\n" +
			"• <code>/setmediadelay &lt;time&gt;</code> — Set delay (1m to 24h)\n" +
			"<b>Time Format:</b>\n" +
			"• <code>5m</code> = 5 minutes\n" +
			"• <code>1h</code> = 1 hour\n" +
			"• <code>12h</code> = 12 hours\n\n" +
			"<b>ℹ️ Note:</b> Applies to photos, videos, stickers, GIFs, Location, Poll and animations.\n" +
			"👮 Only group admins can configure this setting.\n",
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
			_, err = m.Reply("⚠️ Usage:\n<code>/mediadelete on</code> — Enable\n<code>/mediadelete off</code> — Disable")
			return err
		} else {
			status := map[bool]string{true: "Enabled", false: "Disabled"}[settings.Enabled]

			_, err = m.Reply(fmt.Sprintf(
				"📊 <b>MediaDelete Status:</b> %s\n⏱ <b>Delay:</b> %s\n🎯 <b>Scope:</b> All users & admins",
				status,
				utils.FormatDuration(settings.Delay),
			))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply("❌ Invalid option.\nUse <code>/mediadelete on</code> or <code>/mediadelete off</code>")
		return err
	}

	if err := database.SetMediaDeleteEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply("❌ Failed to update setting.")
		return err
	}

	status := "🎬 Media auto-delete enabled ✅"
	if !enable {
		status = "🚫 Media auto-delete disabled ❌"
	}
	_, err = m.Reply(status)
	return err
}

func SetMediaDelayCmd(m *telegram.NewMessage) error {
	var err error
	if _, err = m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err = m.Reply("⚠️ Usage: <code>/setmediadelay 5m</code> or <code>/setmediadelay 2h</code>\n\nValid range: 1m to 24h")
		return err
	}

	duration, err := utils.ParseDuration(m.Args())
	if err != nil {
		_, err = m.Reply("❌ Invalid time format. Use: 5m, 1h, 12h, etc.\nRange: 1 minute to 24 hours")
		return err
	}

	if duration < time.Minute || duration > 24*time.Hour {
		_, err = m.Reply("❌ Delay must be between 1 minute and 24 hours.")
		return err
	}

	if err := database.SetMediaDeleteDelay(m.ChatID(), duration); err != nil {
		_, err = m.Reply("❌ Failed to set delay.")
		return err
	}

	_, err = m.Reply(fmt.Sprintf("✅ Media auto-delete delay set to %s", utils.FormatDuration(duration)))
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
