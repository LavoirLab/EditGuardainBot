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

var startMsG string = `<b>🛡 Hello <a href="tg://user?id=%d">%s</a>!</b> 👋  
I'm <b><a href="tg://user?id=%d">%s</a></b>, your group’s advanced security and management bot.

I can help you with:
✨ <b>Auto-Cleanup</b> of unwanted messages
🚫 <b>Anti-Abuse</b> & Link Protection
🔞 <b>NSFW Detection</b>
🗑 <b>Bulk Purging</b>
📝 <b>Long Message</b> handling

<b>How to use me?</b>
1️⃣ Add me to your group
2️⃣ Grant me admin permissions
3️⃣ I'll start protecting automatically!

👇 <b>Need help?</b> Use the buttons below or send /help.`

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
		m.Delete()

		args := strings.Fields(m.Args())
		if len(args) >= 1 {
			modName := args[0]
			if modName == "help" {
				return handleHelp(m)
			} else if strings.HasPrefix(modName, "info_") {
				userIDStr := strings.TrimPrefix(modName, "info_")
				userID, err := strconv.ParseInt(userIDStr, 10, 64)
				if err == nil {
					peer, err := m.Client.GetPeer(userID)
					if err == nil {
						var (
							Name  string
							ID    int64
							Link1 string
							Link2 string
						)

						switch p := peer.(type) {
						case *telegram.UserObj:
							ID = p.ID
							Name = strings.TrimSpace(p.FirstName + " " + p.LastName)
							Link2 = fmt.Sprintf("tg://openmessage?user_id=%d", ID)
							if p.Username != "" {
								Link1 = fmt.Sprintf("t.me/%s", p.Username)
							} else {
								Link1 = fmt.Sprintf("tg://user?id=%d", ID)
							}
						case *telegram.Channel:
							ID = p.ID
							Name = p.Title
							if p.Username != "" {
								Link1 = "https://t.me/" + p.Username
								Link2 = Link1
							} else {
								Link1 = fmt.Sprintf("https://t.me/c/%d/%d", ID, m.ID)
								Link2 = Link1
							}
						}

						info := fmt.Sprintf(`<b>👤 Entity Information</b>

🆔 <b>ID:</b> <code>%d</code>
👤 <b>Name:</b> %s

🔗 <b>Links:</b>
• <a href="%s">Permanent Link</a>
• <a href="%s">Profile Link</a>`,
							ID, Name, Link1, Link2)

						_, err = m.Reply(info)
						return err
					}
				}
			}

			if module, exists := GetModule(modName); exists &&
				module.Help != "" {
				_, err := m.Reply(module.Help)
				return err
			}
		}

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
		_, err := m.ReplyMedia(
			config.StartImage,
			&telegram.MediaOptions{Caption: caption, ReplyMarkup: btn.Build()},
		)
		return err
	}

	if !m.IsGroup() || m.Channel == nil {
		// Basic group check logic from old watcher/grouponly could go here but watcher handles it now
		return nil
	}

	// Supergroup Chat
	m.Delete()
	database.AddServedChat(m.ChatID())

	btn := telegram.NewKeyboard().AddRow(
		telegram.Button.URL("📖 Help & Commands", fmt.Sprintf("https://t.me/%s?start=help", m.Client.Me().Username)),
	).
		Build()

	caption := fmt.Sprintf(`<b>🛡️ Bot Activated!</b>

I am now active and ready to protect <b>%s</b>.

💡 <b>Next Steps:</b>
• Ensure I am an <b>Admin</b> with <i>Delete Messages</i> permissions.
• Check the available commands by tapping the button below.
• Visit <a href="%s">Support Channel</a> for updates.`, html.EscapeString(m.Chat.Title), config.SupportChannel)

	_, err := m.Reply(caption, &telegram.SendOptions{ReplyMarkup: btn})
	return err
}
