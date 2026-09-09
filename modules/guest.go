package modules

import (
	"fmt"
	"html"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "guestchat",
		ModName: "👤 Guest Chat",
		Help:    "",
		Handlers: []HandlerConfig{
			{
				Pattern: "guestchat",
				Handler: handleGuestChatQuery,
				Group:   1,
			},
		},
	})
}

func handleGuestChatQuery(guest *telegram.GuestChatQuery) error {
	if guest == nil || guest.Client == nil {
		return nil
	}

	if guest.Message != nil &&
		strings.TrimSpace(guest.Message.Text()) != "" {
		return nil
	}

	botID := int64(0)
	botName := "EditGuardian Bot"
	botUsername := ""

	if BotInfo != nil {
		botID = BotInfo.ID
		if BotInfo.FirstName != "" {
			botName = html.EscapeString(BotInfo.FirstName)
		}
		botUsername = BotInfo.Username
	}

	caption := "🛡️ <a href=\"tg://user?id=" + fmt.Sprintf("%d", botID) +
		"\">" + botName + "</a>\n\n" +
		"<i>Group protection, made easy!</i>\n\n" +
		"<blockquote>Add me & stay spam-free 🚫</blockquote>"

	keyboard := telegram.NewKeyboard().
		AddRow(
			telegram.Button.URL(
				"➕ Add Me",
				"https://t.me/"+botUsername+"?startgroup=true",
			),
		).
		Build()

	_, err := guest.Article(
		botName,
		"Click to add me to your group",
		caption,
		&telegram.ArticleOptions{
			ParseMode:   "HTML",
			ReplyMarkup: keyboard,
		},
	)
	return err
}