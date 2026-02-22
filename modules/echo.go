package modules

import (
	"fmt"
	"html"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "echo",
		ModName: "📝 Echo",
		Help: `<b>Command:</b>
<blockquote>/echo &lt;text&gt;
 /setlongmode &lt;off|manual|automatic&gt;
 /setlonglimit &lt;number&gt;</blockquote>

<b>Description:</b>
Sends back the provided text. Also allows setting how the bot handles long messages.

<b>Echo Text:</b>
• <b>/echo</b> &lt;text&gt; – If the message is too long, uploads it to Telegraph and sends as the link.
• <b>/echo</b> &lt;text&gt; (with reply) – Replies with Telegraph link instead of sending normally.

<b>Mode Settings:</b>
• <b>/setlongmode</b> <code>off</code> – No action on long messages.
• <b>/setlongmode</b> <code>manual</code> – Deletes, warns user.
• <b>/setlongmode</b> <code>automatic</code> – Deletes, sends Telegraph link ( Default ).

<b>Custom Limit:</b>
• <b>/setlonglimit</b> <code>&lt;number&gt;</code> – Set character limit (200–4000, default: 800).`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:echo",
				Handler: EcoHandler,
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

func EcoHandler(m *telegram.NewMessage) error {
	if !m.IsGroup() || m.Channel == nil {
		return nil
	}
	if _, err := m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err := m.Reply(
			"⚠️ <b>Usage:</b> <code>/echo &lt;text&gt;</code>\n\nPlease provide the long text you want to share via Telegraph.",
		)
		return err
	}

	settings, err := database.GetEchoSettings(m.ChatID())
	if err != nil {
		_, err := m.Reply(
			"⚠️ <b>Oops! Something went wrong.</b>\n\nI couldn't retrieve the message limit settings for this chat. Please try again.",
		)
		return err
	}

	if len(m.Text()) < settings.Limit {
		isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
		if isadmin {
			_, err := m.Reply(
				fmt.Sprintf(
					"📝 <b>Note:</b> Your message is only <code>%d</code> characters, which is under the <code>%d</code> character limit. As an admin, you can send it normally, but if you'd like a Telegraph link, please send a longer text.",
					len(m.Text()),
					settings.Limit,
				),
			)
			return err
		}
		_, err := m.Reply(
			fmt.Sprintf(
				"📝 <b>Note:</b> Your message is under the <code>%d</code> character limit. You can send it normally without using the <code>/echo</code> command!",
				settings.Limit,
			),
		)
		return err
	}

	text := strings.SplitN(m.Text(), " ", 2)[1]

	return sendEchoMessage(m, text)
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

	if m.Text() == "" {
		return nil
	}
	if len(m.Text()) < settings.Limit {
		return nil
	}

	var isAutomatic bool
	switch settings.Mode {
	case "OFF":
		return nil
	case "AUTO":
		isAutomatic = true
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	if !isAutomatic {
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

💡 <b>Tip:</b> Please shorten your message or use the <code>/echo</code> command to share long texts via Telegraph. 📜
`, id, name, settings.Limit)

			_, _ = m.Reply(text)
			deleteWarningTracker.chats[chatID] = time.Now()
		}
	} else {
		_ = sendEchoMessage(m, m.Text())
	}

	return telegram.ErrEndGroup
}

func sendEchoMessage(m *telegram.NewMessage, text string) error {
	var authorURL string
	userFullName := strings.TrimSpace(
		m.Sender.FirstName + " " + m.Sender.LastName,
	)
	if um := m.Sender.Username; um != "" {
		authorURL = fmt.Sprintf("https://t.me/%s", um)
	} else {
		authorURL = config.SupportChannel
	}

	url, err := utils.CreateTelegraphPage(text, userFullName, authorURL)
	if err != nil {
		return err
	}

	var msg string
	opts := telegram.SendOptions{
		LinkPreview: false,
	}

	if m.IsReply() {
		rmsg, err := m.Client.GetMessageByID(m.ChatID(), m.ReplyID())
		if err == nil && rmsg.Sender != nil {
			replyUserFullName := strings.TrimSpace(
				rmsg.Sender.FirstName + " " + rmsg.Sender.LastName,
			)

			// Use the full template with both names
			msg = fmt.Sprintf(
				`👋 Hello <a href="tg://user?id=%d">%s</a>,

<a href="tg://user?id=%d">%s</a> <b>wanted to share a long message with you.</b> 📝

📖 <b>View full message on:</b> <a href="%s">Telegraph</a>`,
				m.ReplySenderID(), html.EscapeString(replyUserFullName),
				m.SenderID(), html.EscapeString(userFullName),
				url,
			)
			opts.ReplyID = m.ReplyID()
		}
	}

	// Non-reply fallback (no empty <a> tag)
	if msg == "" {
		msg = fmt.Sprintf(
			`👋 <a href="tg://user?id=%d">%s</a> <b>wanted to share a long message.</b> 📝

📖 <b>View full message on:</b> <a href="%s">Telegraph</a>`,
			m.SenderID(),
			html.EscapeString(userFullName),
			url,
		)
	}

	_, err = m.Reply(msg, &opts)
	return err
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
