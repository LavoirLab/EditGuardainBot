package modules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/utils"
	"main/database"
)

var startMsG string = `<b>🛡 Hello <a href="tg://user?id=%d">%s</a>!</b> 👋  
I'm <b><a href="tg://user?id=%d">%s</a></b>, your group’s security bot keeping chats clean and safe.

📣 Stay informed with instant alerts.  
✅ Add me now and I’ll start protecting your group!`

func startCB(c *telegram.CallbackQuery) error {
	c.Answer("")
	cl := c.Client.Me()
	caption := fmt.Sprintf(startMsG,
		c.Sender.ID, strings.TrimSpace(c.Sender.FirstName+" "+c.Sender.LastName),
		cl.ID, cl.FirstName)

	btn := telegram.NewKeyboard()
	btn.AddRow(
		telegram.Button.URL("🔄 Update Channel", config.SupportChannel),
		telegram.Button.URL("💬 Update Group", config.SupportChat),
	)
	btn.AddRow(
		telegram.Button.Data("❓ Help & Commands", "help"),
	)
	btn.AddRow(
		telegram.Button.URL("➕ Add me to Your Group",
			fmt.Sprintf("https://t.me/%s?startgroup=s&admin=delete_messages+invite_users", c.Client.Me().Username),
		),
	)

	replyMarkup := btn.Build()
	c.Edit(caption, &telegram.SendOptions{
		ReplyMarkup: replyMarkup,
	})
	return telegram.EndGroup
}

func start(m *telegram.NewMessage) error {
	if m.ChatType() == telegram.EntityUser {
		m.Delete()

		args := strings.Fields(m.Text())
		if len(args) >= 2 {
			modName := args[1]
			if strings.HasPrefix(modName, "help") {
				return help(m)
			} else if strings.HasPrefix(modName, "info_") {
				userIDStr := strings.TrimPrefix(modName, "info_")
				userID, err := strconv.ParseInt(userIDStr, 10, 64)
				if err != nil {
					return err
				}

				peer, err := m.Client.GetPeer(userID)
				if err != nil {
					return err
				}

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

				info := fmt.Sprintf(`
Name: %s
Id: %d
Link: <a href="%s">Link 1</a> <a href="%s">Link 2</a>`,
					Name, ID, Link1, Link2)

				return m.E(m.Respond(info))
			}

			if h := GetHelp(modName); h != "" {
				return m.E(m.Respond(h))
			}
		}

		userFullName := strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
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
			telegram.Button.Data("❓ Help & Commands", "help"),
		)
		btn.AddRow(
			telegram.Button.URL("➕ Add me to Your Group",
				fmt.Sprintf("https://t.me/%s?startgroup=s&admin=delete_messages+invite_users", m.Client.Me().Username),
			),
		)

		replyMarkup := btn.Build()

		database.AddServedUser(m.Sender.ID)
		return m.E(m.RespondMedia(config.StartImage, telegram.MediaOptions{Caption: caption, ReplyMarkup: replyMarkup}))

	}

	if IsBasicGroup(m) {
		msg := `⚠️ Warning: I can't function in a basic group!

To use my features, please upgrade this group to a supergroup.

✅ How to upgrade:
1. Go to Group Settings.
2. Tap on "Chat History" and set it to "Visible".
3. Re-add me, and I'll be ready to help!`

		m.Respond(msg)
		m.Delete()
		m.Client.LeaveChannel(m.ChatID(), true)
	}

	// Supergroup Chat
	if IsSupergroup(m) {
		m.Delete()
		database.AddServedChat(m.ChatID())
		if !utils.WarnIfLackOfPms(m.Client, m, m.ChatID()) {
			return nil
		}

		return m.E(m.Respond("✅ I am active and ready to protect this supergroup!"))
	}

	return nil
}
