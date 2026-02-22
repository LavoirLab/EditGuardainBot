package modules

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "longmode",
		ModName: "📏 LongMode",
		Help: `<b>Commands:</b>
• /setlonglimit <code>&lt;number&gt;</code> – Set character limit (200–4000).
• /nolongmessage <code>&lt;on|off&gt;</code> – Enable or disable long message protection.

<b>Description:</b>
Automatically deletes messages that exceed the allowed character limit and warns the user.`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:setlonglimit",
				Handler: RequirePermission("change_info", SetLongLimitHandler),
			},
			{
				Pattern: "command:nolongmessage",
				Handler: RequirePermission("delete_messages", NoLongMessageCmd),
			},
		},
	})
}

type warningTracker struct {
	globalLock sync.Mutex            // Protects access to locks map
	locks      map[int64]*sync.Mutex // Per-chat locks
	chats      map[int64]time.Time   // Last warning per chat
}

var deleteWarningTracker = warningTracker{
	locks: make(map[int64]*sync.Mutex),
	chats: make(map[int64]time.Time),
}

func (w *warningTracker) Lock(chatId int64) {
	w.globalLock.Lock()
	lock, exists := w.locks[chatId]
	if !exists {
		lock = &sync.Mutex{}
		w.locks[chatId] = lock
	}
	w.globalLock.Unlock()

	lock.Lock()
}

func (w *warningTracker) Unlock(chatId int64) {
	w.globalLock.Lock()
	lock := w.locks[chatId]
	w.globalLock.Unlock()

	if lock != nil {
		lock.Unlock()
	}
}

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
		_, err := m.Reply(fmt.Sprintf("📏 <b>Long Message Protection</b>\n<b>Status:</b> <i>%s</i>\n\nUsage: <code>/nolongmessage &lt;on|off&gt;</code>", status))
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err := m.Reply("❌ <b>Invalid usage.</b>\n\nPlease use <code>/nolongmessage on</code> or <code>/nolongmessage off</code>.")
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

	_, err = m.Reply(fmt.Sprintf("✅ <b>Long message protection %s!</b>", status))
	return err
}

func LongMessageHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
		return nil
	}

	chatID := m.ChatID()

	isadmin, _ := utils.IsAdmin(m.Client, chatID, m.SenderID())
	if isadmin {
		return nil
	}

	settings, err := database.GetEchoSettings(chatID)
	if err != nil {
		return nil
	}

	if settings.Mode == "OFF" {
		return nil
	}

	if m.Text() == "" {
		return nil
	}
	if len(m.Text()) < settings.Limit {
		return nil
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	deleteWarningTracker.Lock(chatID)
	defer deleteWarningTracker.Unlock(chatID)

	lastWarning, exists := deleteWarningTracker.chats[chatID]
	if !exists || time.Since(lastWarning) > time.Second {
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

⚠️ <a href="tg://user?id=%d">%s</a>, <i>your message exceeds the %d-character limit allowed in this group.</i> 🚫

💡 <b>Tip:</b> Please shorten your message or send it in multiple parts to stay within the limit. 📜
`, id, name, settings.Limit)

		_, _ = m.Reply(text)
		deleteWarningTracker.chats[chatID] = time.Now()
	}

	return telegram.ErrEndGroup
}
