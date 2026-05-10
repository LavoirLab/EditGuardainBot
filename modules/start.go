package modules

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/database"
)

var startMsG string = `<emoji id='5251203410396458957'>🛡</emoji> <b>Hᴇʟʟᴏ <a href="tg://user?id=%d">%s</a>!</b> <emoji id='5440422734402173876'>👋</emoji>

<i>ɪ'ᴍ <b><a href="tg://user?id=%d">%s</a></b>, ʏᴏᴜʀ ɢʀᴏᴜᴘ’ꜱ ᴀᴅᴠᴀɴᴄᴇᴅ ꜱᴇᴄᴜʀɪᴛʏ ᴀɴᴅ ᴍᴀɴᴀɢᴇᴍᴇɴᴛ ʙᴏᴛ.</i>

<b>ɪ ᴄᴀɴ ʜᴇʟᴘ ʏᴏᴜ ᴡɪᴛʜ:</b>
<blockquote><i><emoji id='5201914481671682382'>✨</emoji> <b>ᴀᴜᴛᴏ-ᴄʟᴇᴀɴᴜᴘ</b> ᴏꜰ ᴜɴᴡᴀɴᴛᴇᴅ ᴍᴇꜱꜱᴀɢᴇꜱ
<emoji id='5462882007451185227'>🚫</emoji> <b>ᴀɴᴛɪ-ᴀʙᴜꜱᴇ</b> & ʟɪɴᴋ ᴘʀᴏᴛᴇᴄᴛɪᴏɴ
<emoji id='5429139425818854423'>🔞</emoji> <b>ɴꜱꜰᴡ ᴅᴇᴛᴇᴄᴛɪᴏɴ</b>
<emoji id='5445267414562389170'>🗑</emoji> <b>ʙᴜʟᴋ ᴘᴜʀɢɪɴɢ</b></i></blockquote>

<emoji id='5411322793373477644'>👇</emoji> <b>ɴᴇᴇᴅ ʜᴇʟᴘ?</b> ᴜꜱᴇ ᴛʜᴇ ʙᴜᴛᴛᴏɴꜱ ʙᴇʟᴏᴡ ᴏʀ ꜱᴇɴᴅ /help.`

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "start",
		ModName: "🚀 Start",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:start",
				Handler: handleStart,
			},
			{
				Pattern: "callback:start_callback",
				Handler: handleStartCallback,
			},
		},
	})
}

func handleStartCallback(c *telegram.CallbackQuery) error {
	c.Answer("")
	cl := c.Client.Me()
	userFullName := strings.TrimSpace(
		c.Sender.FirstName + " " + c.Sender.LastName,
	)
	caption := fmt.Sprintf(startMsG,
		c.Sender.ID, userFullName,
		cl.ID, cl.FirstName)

	btn := telegram.NewKeyboard()
	btn.AddRow(
		telegram.Button.URL("🔄 Update Channel", config.SupportChannel),
		telegram.Button.URL("💬 Update Group", config.SupportChat),
	)
	btn.AddRow(
		telegram.Button.Data("📖 Command Help", "help"),
	)
	btn.AddRow(
		telegram.Button.URL(
			"➕ Add me to Your Group",
			fmt.Sprintf(
				"https://t.me/%s?startgroup=s&admin=delete_messages+invite_users",
				c.Client.Me().Username,
			),
		),
	)

	_, err := c.Edit(caption, &telegram.SendOptions{
		ReplyMarkup: btn.Build(),
	})
	return err
}

func handleStart(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		handled, err := handleDeepLink(m)
		if err != nil || handled {
			return err
		}
		return sendPrivateStart(m)
	}

	if m.IsGroup() && m.Channel != nil {
		return handleGroupStart(m)
	}

	return nil
}

func handleDeepLink(m *telegram.NewMessage) (bool, error) {
	args := strings.Fields(m.Args())
	if len(args) < 1 {
		return false, nil
	}

	modName := args[0]
	if modName == "help" {
		return true, handleHelp(m)
	}

	if module, exists := GetModule(modName); exists && module.Help != "" {
		_, err := m.Reply(module.Help)
		return true, err
	}

	if strings.HasPrefix(modName, "nsfwlist_") {
		chatIDStr := strings.TrimPrefix(modName, "nsfwlist_")
		chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
		if err == nil {
			return true, HandleNsfwListDeepLink(m, chatID)
		}
	}

	if strings.HasPrefix(modName, "blockedlist_") {
		chatIDStr := strings.TrimPrefix(modName, "blockedlist_")
		chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
		if err == nil {
			return true, HandleBlockedListDeepLink(m, chatID)
		}
	}

	return false, nil
}

func logUserStart(m *telegram.NewMessage, userFullName string) {
	userMention := fmt.Sprintf(
		"<a href=\"tg://user?id=%d\">%s</a>",
		m.Sender.ID,
		html.EscapeString(userFullName),
	)
	if m.Sender.Username != "" {
		userMention = fmt.Sprintf(
			"%s (@%s)",
			userMention,
			m.Sender.Username,
		)
	}

	logMsg := fmt.Sprintf(
		"👤 <b>User Started Bot</b>\n"+
			"━━━━━━━━━━━━━━━━━\n"+
			"👤 <b>User:</b> %s\n"+
			"🆔 <b>ID:</b> <code>%d</code>\n"+
			"━━━━━━━━━━━━━━━━━",
		userMention,
		m.Sender.ID,
	)
	_, _ = m.Client.SendMessage(config.LoggerId, logMsg)
}

func sendPrivateStart(m *telegram.NewMessage) error {
	userFullName := strings.TrimSpace(
		m.Sender.FirstName + " " + m.Sender.LastName,
	)
	botName := strings.TrimSpace(m.Client.Me().FirstName)

	caption := fmt.Sprintf(startMsG,
		m.Sender.ID, userFullName,
		m.Client.Me().ID, botName)

	btn := telegram.NewKeyboard()
	btn.AddRow(
		telegram.Button.URL("🔄 Update Channel", config.SupportChannel),
		telegram.Button.URL("💬 Update Group", config.SupportChat),
	)
	btn.AddRow(
		telegram.Button.Data("📖 Command Help", "help"),
	)
	btn.AddRow(
		telegram.Button.URL(
			"➕ Add me to Your Group",
			fmt.Sprintf(
				"https://t.me/%s?startgroup=s&admin=delete_messages+invite_users",
				m.Client.Me().Username,
			),
		),
	)

	database.AddServedUser(m.Sender.ID)
	logUserStart(m, userFullName)

	_, err := m.ReplyMedia(
		config.StartImage,
		&telegram.MediaOptions{Caption: caption, ReplyMarkup: btn.Build()},
	)
	return err
}

func handleGroupStart(m *telegram.NewMessage) error {
	m.Delete()
	database.AddServedChat(m.ChatID())

	btn := telegram.NewKeyboard().AddRow(
		telegram.Button.URL("📖 Help & Commands", fmt.Sprintf("https://t.me/%s?start=help", m.Client.Me().Username)),
	).
		Build()

x := ""

if m.Channel != nil {
x = m.Channel.Title
} else if m.Chat != nil {
x = m.Chat.Title
}

if x == "" {
x = "this chat"
}
	caption := fmt.Sprintf(`<b>🛡️ Bot Activated!</b>

I am now active and ready to protect <b>%s</b>.`, html.EscapeString(x))

	_, err := m.Reply(caption, &telegram.SendOptions{ReplyMarkup: btn})
	return err
}
