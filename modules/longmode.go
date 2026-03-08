package modules

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "longmode",
		ModName: "📏 LongMode",
		Help: `<i>ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇʟᴇᴛᴇs ᴍᴇssᴀɢᴇs ᴛʜᴀᴛ ᴇxᴄᴇᴇᴅ ᴛʜᴇ ᴀʟʟᴏᴡᴇᴅ ᴄʜᴀʀᴀᴄᴛᴇʀ ʟɪᴍɪᴛ ᴀɴᴅ ᴡᴀʀɴs ᴛʜᴇ ᴜsᴇʀ.<\i>

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• /setlonglimit &lt;number&gt; – sᴇᴛ ᴄʜᴀʀᴀᴄᴛᴇʀ ʟɪᴍɪᴛ (200–4000).
• /nolongmessage &lt;on|off&gt; – ᴇɴᴀʙʟᴇ ᴏʀ ᴅɪsᴀʙʟᴇ ʟᴏɴɢ ᴍᴇssᴀɢᴇ ᴘʀᴏᴛᴇᴄᴛɪᴏɴ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:setlonglimit",
				Handler: RequirePermission(
					"delete_messages",
					SetLongLimitHandler,
				),
			},
			{
				Pattern: "command:nolongmessage",
				Handler: RequirePermission("delete_messages", NoLongMessageCmd),
			},
		},
	})
}

type longModeCacheKey struct {
	chatID int64
	userID int64
}

// longModeWarningCache throttles warning messages to once per second per user per chat,
// preventing spam while ensuring individual users are warned.
var longModeWarningCache = utils.NewCache[longModeCacheKey, bool](time.Second)

func SetLongLimitHandler(m *telegram.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err := m.Reply(
			"⚠️ <b>Usage:</b> <code>/setlonglimit &lt;number&gt;</code>\n\n<b>Example:</b> <code>/setlonglimit 1000</code>\n\n<i>Valid range: 200 to 4000 characters.</i>",
		)
		return err
	}

	args := strings.Fields(m.Args())
	limit, convErr := strconv.Atoi(args[0])
	if convErr != nil || limit < 200 || limit > 4000 {
		_, err := m.Reply(
			"❌ <b>Invalid limit.</b>\n\nPlease provide a number between <code>200</code> and <code>4000</code>.",
		)
		return err
	}

	err := database.SetEchoLimit(m.ChatID(), limit)
	if err != nil {
		_, err = m.Reply(
			"⚠️ <b>Failed to save limit.</b>\n\nSomething went wrong while updating the character limit. Please try again.",
		)
		return err
	}

	_, err = m.Reply(
		fmt.Sprintf(
			"✅ <b>Long message limit set to %d characters!</b>\n\nMessages exceeding this length will be automatically deleted and a warning will be issued.",
			limit,
		),
	)
	return err
}

func NoLongMessageCmd(m *telegram.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}

	args := strings.Fields(m.Args())
	if len(args) == 0 {
		settings, _ := database.GetEchoSettings(m.ChatID())
		status := "Enabled"
		if settings.Mode == "OFF" {
			status = "Disabled"
		}
		_, err := m.Reply(
			fmt.Sprintf(
				"📏 <b>Long Message Protection</b>\n<b>Status:</b> <i>%s</i>\n\nUsage: <code>/nolongmessage &lt;on|off&gt;</code>",
				status,
			),
		)
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err := m.Reply(
			"❌ <b>Invalid usage.</b>\n\nPlease use <code>/nolongmessage on</code> or <code>/nolongmessage off</code>.",
		)
		return err
	}

	err = database.SetLongMessageStatus(m.ChatID(), enable)
	if err != nil {
		_, err = m.Reply("⚠️ <b>Update failed.</b>\n\nPlease try again later.")
		return err
	}

	status := "enabled"
	if !enable {
		status = "disabled"
	}

	_, err = m.Reply(
		fmt.Sprintf("✅ <b>Long message protection %s!</b>", status),
	)
	return err
}

func LongMessageHandler(m *telegram.NewMessage) error {
	// Early return for empty or very short messages (fast path)
	// 200 is the minimum configurable limit in SetLongLimitHandler
	if m.Text() == "" || len(m.Text()) < 200 {
		return nil
	}

	if m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	chatID := m.ChatID()

	// Move settings check before admin check to skip lookup if feature is off
	settings, err := database.GetEchoSettings(chatID)
	if err != nil || settings.Mode == "OFF" {
		return nil
	}

	// Final length check against configured limit
	if len(m.Text()) < settings.Limit {
		return nil
	}

	// Expensive admin check only for messages that actually violate the limit
	isadmin, _ := utils.IsAdmin(m.Client, chatID, m.SenderID())
	if isadmin {
		return nil
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	// Non-blocking warning throttle using cache (per-user per-chat)
	cacheKey := longModeCacheKey{chatID: chatID, userID: m.SenderID()}
	if _, warned := longModeWarningCache.Get(cacheKey); !warned {
		var name string
		var id int64
		if m.SenderChat != nil && m.SenderChat.ID != 0 {
			name = m.SenderChat.Title
			id = m.SenderChat.ID
		} else if m.Sender != nil {
			name = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
			id = m.Sender.ID
		}

		text := fmt.Sprintf(`
📏 <b>Message Too Long!</b>

⚠️ <a href="tg://user?id=%d">%s</a>, <i>your message exceeds the %d-character limit allowed in this group.</i> 🚫`, id, name, settings.Limit)

		_, _ = m.Reply(text)
		longModeWarningCache.Set(cacheKey, true)
	}

	return telegram.ErrEndGroup
}
