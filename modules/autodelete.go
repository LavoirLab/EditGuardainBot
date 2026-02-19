// file: modules/msgautodelete.go - FIXED VERSION
package modules

import (
	"fmt"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
	"main/database"
)

func init() {
	AddHelp(
		"💬 MsgDelete",
		"msgdelete_help",
		"💬 <b>Message Auto-Delete</b> automatically removes all messages after a set delay.\n\n"+
			"<b>Commands:</b>\n"+
			"• <code>/msgdelete on</code> — Enable message auto-delete ✅\n"+
			"• <code>/msgdelete off</code> — Disable message auto-delete ❌\n"+
			"• <code>/setmsgdelay &lt;time&gt;</code> — Set delay (1m to 12h)\n\n"+
			"<b>Time Format:</b>\n"+
			"• <code>5m</code> = 5 minutes\n"+
			"• <code>1h</code> = 1 hour\n"+
			"• <code>12h</code> = 12 hours\n\n"+
			"<b>ℹ️ Note:</b> Applies to all text and media messages from regular users only.\n"+
			"👮 Only group admins can configure this setting.",
	)
}

func MsgDeleteCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Text())
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> msgdelete -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> msgdelete -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")
		return telegram.EndGroup
	}

	if len(args) < 2 {
		settings, err := database.GetMsgDeleteSettings(m.ChatID())
		if err != nil {
			m.Respond("⚠️ Usage:\n<code>/msgdelete on</code> — Enable\n<code>/msgdelete off</code> — Disable")
		} else {
			status := map[bool]string{true: "Enabled", false: "Disabled"}[settings.Enabled]
			m.Respond(fmt.Sprintf("📊 <b>MsgDelete Status:</b> %s\n⏱ <b>Delay:</b> %s\n🛡️ <b>Note:</b> Admins are always exempt", status, formatDuration(settings.Delay)))
		}
		return telegram.EndGroup
	}

	arg := strings.ToLower(args[1])
	enable := arg == "on"
	if arg != "on" && arg != "off" {
		m.Respond("❌ Invalid option.\nUse <code>/msgdelete on</code> or <code>/msgdelete off</code>")
		return telegram.EndGroup
	}

	if err := database.SetMsgDeleteEnabled(m.ChatID(), enable); err != nil {
		m.Respond("❌ Failed to update setting.")
		return L(m, "Modules -> msgdelete -> SetMsgDeleteEnabled", err)
	}

	status := "💬 Message auto-delete enabled ✅"
	if !enable {
		status = "🚫 Message auto-delete disabled ❌"
	}
	m.Respond(status)
	return telegram.EndGroup
}

func SetMsgDelayCmd(m *telegram.NewMessage) error {
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> msgdelete -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> msgdelete -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")
		return telegram.EndGroup
	}

	if m.Args() == "" {
		m.Respond("⚠️ Usage: <code>/setmsgdelay 5m</code> or <code>/setmsgdelay 2h</code>\n\nValid range: 1m to 12h")
		return telegram.EndGroup
	}

	duration, err := parseDuration(m.Args())
	if err != nil {
		m.Respond("❌ Invalid time format. Use: 5m, 1h, 12h, etc.\nRange: 1 minute to 12 hours")
		return telegram.EndGroup
	}

	if duration < time.Minute || duration > 12*time.Hour {
		m.Respond("❌ Delay must be between 1 minute and 12 hours.")
		return telegram.EndGroup
	}

	if err := database.SetMsgDeleteDelay(m.ChatID(), duration); err != nil {
		m.Respond("❌ Failed to set delay.")
		return L(m, "Modules -> msgdelete -> SetMsgDeleteDelay", err)
	}

	m.Respond(fmt.Sprintf("✅ Message auto-delete delay set to %s", formatDuration(duration)))
	return telegram.EndGroup
}

func handleMsgAutoDelete(m *telegram.NewMessage) error {
	if !IsSupergroup(m) {
		return nil
	}

	settings, err := database.GetMsgDeleteSettings(m.ChatID())
	if err != nil || !settings.Enabled {
		return nil
	}

	ScheduleMessageDeletion(m.Client, m.ChatID(), int32(m.ID), settings.Delay)

	return telegram.EndGroup
}
