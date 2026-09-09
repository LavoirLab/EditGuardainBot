package modules

import (
	"fmt"
	"html"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/utils"
)

const joinRequestPhotoURL = "https://files.catbox.moe/472piq.jpg"

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "joinrequest",
		ModName: "📥 Join Requests",
		Help:    "",
		Handlers: []HandlerConfig{
			{
				Pattern: "joinrequest",
				Handler: handleJoinRequest,
				Group:   1,
			},
		},
	})
}

func joinRequestChatTitle(jr *telegram.JoinRequestUpdate) string {
	if jr.Channel != nil {
		return jr.Channel.Title
	}
	if jr.Chat != nil {
		return jr.Chat.Title
	}
	return "the group"
}

func joinRequestBotMention() string {
	if BotInfo == nil {
		return "your Group Protection Bot 🛡️"
	}

	name := "Group Protection Bot"
	if BotInfo.FirstName != "" {
		name = html.EscapeString(BotInfo.FirstName)
	}

	return fmt.Sprintf(
		"<a href=\"tg://user?id=%d\">%s</a>",
		BotInfo.ID,
		name,
	)
}

func handleJoinRequest(jr *telegram.JoinRequestUpdate) error {
	if jr == nil || jr.Client == nil || jr.IsEmpty() {
		return nil
	}

	for _, user := range jr.Users {
		if user == nil {
			continue
		}

		caption := fmt.Sprintf(
			"✦ <b>Hey %s 👋</b>\n"+
				"━━━━━━━━━━━━━━━━━━━━\n\n"+
				"<i>Thanks for requesting to join <b>%s</b>!</i>\n\n"+
				"<b>🛡️ I'm %s — A Powerful Group Protection & Moderation Bot</b>\n\n"+
				"<blockquote>• Blocks spam, links & promos 🚫\n"+
				"• Filters NSFW content 🔞\n"+
				"• Stops bot promotions 🤖\n"+
				"• Removes abusive language 💬</blockquote>\n\n"+
				"Send <b>/start</b> to see my menu & commands 📋",
			utils.MentionHTML(user),
			html.EscapeString(joinRequestChatTitle(jr)),
			joinRequestBotMention(),
		)

		botUsername := ""
		if BotInfo != nil {
			botUsername = BotInfo.Username
		}

		keyboard := telegram.NewKeyboard().
			AddRow(
				telegram.Button.URL(
					"➕ Add Me In Your Group",
					fmt.Sprintf("https://t.me/%s?startgroup=true", botUsername),
				),
			).
			AddRow(
				telegram.Button.URL("💬 Support", config.SupportChat),
				telegram.Button.URL("📢 Updates", config.SupportChannel),
			).
			Build()

		_, err := jr.Client.SendMedia(user.ID, joinRequestPhotoURL, &telegram.MediaOptions{
			Caption:     caption,
			ParseMode:   "HTML",
			ReplyMarkup: keyboard,
		})
		if err != nil {
			jr.Client.Log.Error(
				fmt.Sprintf("Failed to send welcome to user %d: %v", user.ID, err),
			)
		}
	}

	return telegram.ErrEndGroup
}