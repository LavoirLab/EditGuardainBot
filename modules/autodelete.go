package modules

import (
	"fmt"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func getMsgDeleteDashboard(enabled bool, delay time.Duration) (string, telegram.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("💬 <b>Message Auto-Delete</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n"+
		"<b>Delay:</b> <code>%s</code>\n\n"+
		"<blockquote expandable>"+
		"When enabled, I automatically remove all group messages after a configured delay. This keeps your chat clean and lightweight by removing old clutter.\n\n"+
		"<b>Action:</b> All text and media messages from regular users are scheduled for deletion after the set time."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> Use <code>/setmsgdelay &lt;time&gt;</code> (e.g., <code>/setmsgdelay 10m</code>) to change the deletion interval.", status, utils.FormatDuration(delay))

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data(toggleLabel, "autodelete_toggle:"+toggleVal),
		telegram.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard
}

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
				Handler: RequirePermission("delete_messages", MsgDeleteCmd),
			},
			{
				Pattern: "command:setmsgdelay",
				Handler: RequirePermission("delete_messages", SetMsgDelayCmd),
			},
			{
				Pattern: "callback:^autodelete_toggle:(on|off)$",
				Handler: handleMsgDeleteToggle,
			},
			{
				Pattern: "*",
				Handler: updateReadChannelOutboxHandler,
			},
		},
	})
}

func handleMsgDeleteToggle(c *telegram.CallbackQuery) error {
	if isAdmin, _ := utils.IsAdmin(c.Client, c.ChatID, c.SenderID); !isAdmin {
		c.Answer("👤 Admin Access Required", &telegram.CallbackOptions{Alert: true})
		return telegram.ErrEndGroup
	}

	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	if err := database.SetMsgDeleteEnabled(c.ChatID, enable); err != nil {
		c.Answer("⚠️ Update failed.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	settings, _ := database.GetMsgDeleteSettings(c.ChatID)
	text, keyboard := getMsgDeleteDashboard(enable, settings.Delay)
	c.Answer("✓ Settings Updated")
	_, err := c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func MsgDeleteCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		settings, _ := database.GetMsgDeleteSettings(m.ChatID())
		text, keyboard := getMsgDeleteDashboard(settings.Enabled, settings.Delay)
		_, err = m.Reply(text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
		return err
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

	settings, _ := database.GetMsgDeleteSettings(m.ChatID())
	text, keyboard := getMsgDeleteDashboard(enable, settings.Delay)
	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
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
	if m.IsChannelPost() {
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
