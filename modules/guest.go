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
			},
		},
	})
}

func handleGuestChatQuery(guest *telegram.GuestChatQuery) error {
	if guest == nil || guest.Client == nil {
		return nil
	}

	botUsername := ""
	if BotInfo != nil {
		botUsername = BotInfo.Username
	}

	if guest.Message != nil {
		queryText := strings.TrimSpace(guest.Message.Text())
		queryText = strings.TrimSpace(
			strings.TrimPrefix(queryText, "@"+botUsername),
		)
		if queryText != "" {
			return nil
		}
	}

	botID := int64(0)
	botName := "EditGuardian Bot"

	if BotInfo != nil {
		botID = BotInfo.ID
		if BotInfo.FirstName != "" {
			botName = html.EscapeString(BotInfo.FirstName)
		}
	}

	caption := "🛡️ <b><a href=\"tg://user?id=" + fmt.Sprintf("%d", botID) +
		"\">" + botName + "</a></b>\n\n" +
		"<i>Your group's all-in-one bodyguard.</i>\n\n" +
		"<blockquote>🚫 Kicks spammers & deletes bad words\n" +
		"🛡️ Guards against raids & floods\n" +
		"⚡ 100% automatic, works 24/7</blockquote>\n\n" +
		"Add me & take a back seat. 😎"

	keyboard := telegram.NewKeyboard().
		AddRow(
			telegram.Button.URL(
				"🛡️ Add Me & Protect Your Group",
				"https://t.me/"+botUsername+"?startgroup=true",
			),
		).
		Build()

	_, err := guest.Article(
		botName,
		"Add me to your group & stay spam-free",
		caption,
		&telegram.ArticleOptions{
			ParseMode:   "HTML",
			ReplyMarkup: keyboard,
		},
	)
	return err
}