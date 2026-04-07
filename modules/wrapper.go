package modules

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func RequirePermission(
	permission string,
	next func(*telegram.NewMessage) error,
) func(*telegram.NewMessage) error {
	return func(m *telegram.NewMessage) error {
		if m.IsChannel() {
			return telegram.ErrEndGroup
		}

		if m.IsPrivate() {
			btn := telegram.NewKeyboard().AddRow(
				telegram.Button.Data("🏠 Back to Menu", "start_callback"),
			).Build()
			_, err := m.Reply(
				"👋 <b>Group Only!</b>\n\n"+
					"<i>This command is designed for group chats.</i>\n\n"+
					"💡 <b>Tip:</b> Please use this command inside a group where I am an administrator.",
				&telegram.SendOptions{
					ReplyMarkup: btn,
					ParseMode:   "HTML",
				},
			)
			if err != nil {
				return err
			}
			return telegram.ErrEndGroup
		}

		if m.Channel == nil {
			return nil
		}

		// Anonymous admin check (inline)
		if m.SenderID() == m.ChatID() || m.SenderID() == 0 {
			return handleAnonAdmin(m, permission, next)
		}

		if m.Sender == nil {
			return telegram.ErrEndGroup
		}

		chatID := m.ChannelID()
		userID := m.SenderID()

		userHasPerm, userIsAdmin, err := utils.CheckUserPermission(
			m.Client, chatID, userID, permission,
		)
		if err != nil {
			log.Printf("Failed to check user permission: %v", err)
			_, err = m.Reply(
				"⚠️ <b>Verification Failed</b>\n\n"+
					"<i>I encountered an error while checking your permissions.</i>\n\n"+
					"💡 <b>Tip:</b> Please try again. If the issue persists, contact the bot owner.",
				&telegram.SendOptions{ParseMode: "HTML"},
			)
			return err
		}

		if !userIsAdmin {
			_, err = m.Reply(
				"👤 <b>Admin Access Required</b>\n\n"+
					"<i>Sorry, only group administrators can use this command.</i>\n\n"+
					"💡 <b>Tip:</b> Contact a group admin if you believe this is an error.",
				&telegram.SendOptions{ParseMode: "HTML"},
			)
			if err != nil {
				return err
			}
			return telegram.ErrEndGroup
		}

		if !userHasPerm {
			_, err = m.Reply(
				fmt.Sprintf(
					"🚫 <b>Permission Denied</b>\n\n"+
						"<i>You need the <code>%s</code> permission to use this command.</i>\n\n"+
						"💡 <b>Tip:</b> Ask a group owner to grant you this specific right.",
					utils.PrettifyPermission(permission),
				),
				&telegram.SendOptions{ParseMode: "HTML"},
			)
			if err != nil {
				return err
			}
			return telegram.ErrEndGroup
		}

		botHasPerm, botIsAdmin, err := utils.CheckUserPermission(
			m.Client, chatID, m.Client.Me().ID, permission,
		)
		if err != nil {
			m.Client.Log.Error(
				fmt.Sprintf("Failed to check bot permission: %v", err),
			)
			_, err = m.Reply(
				"⚠️ <b>Verification Failed</b>\n\n"+
					"<i>I couldn't verify my own permissions in this chat.</i>\n\n"+
					"💡 <b>Tip:</b> Please try running the command again in a few moments.",
				&telegram.SendOptions{ParseMode: "HTML"},
			)
			return err
		}

		if !botIsAdmin {
			_, err = m.Reply(
				"🤖 <b>I am not an Admin!</b>\n\n"+
					"<i>I need administrator rights to perform this action.</i>\n\n"+
					"💡 <b>Tip:</b> Please promote me to Admin and ensure I have the <i>Delete Messages</i> permission.",
				&telegram.SendOptions{ParseMode: "HTML"},
			)
			if err != nil {
				return err
			}
			return telegram.ErrEndGroup
		}

		if !botHasPerm {
			_, err = m.Reply(
				fmt.Sprintf(
					"🚫 <b>Missing Bot Permission</b>\n\n"+
						"<i>I don't have the <code>%s</code> permission.</i>\n\n"+
						"💡 <b>Tip:</b> Please grant me this permission so I can assist you.",
					utils.PrettifyPermission(permission),
				),
				&telegram.SendOptions{ParseMode: "HTML"},
			)
			if err != nil {
				return err
			}
			return telegram.ErrEndGroup
		}
		return next(m)
	}
}

var anonVerifyCache = utils.NewCache[string, *AnonVerification](5 * time.Minute)

type AnonVerification struct {
	OriginalMessage *telegram.NewMessage
	Permission      string
	Handler         func(*telegram.NewMessage) error
	CreatedAt       time.Time
}

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "anon_verification",
		ModName: "Anonymous Admin Verification",
		Handlers: []HandlerConfig{
			{
				Pattern: "callback:^anon_verify:",
				Handler: HandleAnonVerification,
			},
		},
	})
}

func handleAnonAdmin(
	m *telegram.NewMessage,
	permission string,
	handler func(*telegram.NewMessage) error,
) error {
	verifyKey := fmt.Sprintf(
		"anon_%d_%d",
		m.ChannelID(),
		time.Now().UnixNano(),
	)

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data("✅ Verify Identity", "anon_verify:"+verifyKey),
	).Build()

	_, err := m.Reply(
		"👤 <b>Identity Verification</b>\n\n"+
			"<i>You are appearing as an anonymous administrator.</i>\n\n"+
			"💡 <b>Tip:</b> Tap the button below to verify your identity and execute the command.",
		&telegram.SendOptions{ReplyMarkup: keyboard, ParseMode: "HTML"},
	)
	if err != nil {
		log.Printf("Failed to send anon verification message: %v", err)
		return err
	}

	anonVerifyCache.Set(verifyKey, &AnonVerification{
		OriginalMessage: m,
		Permission:      permission,
		Handler:         handler,
		CreatedAt:       time.Now(),
	})

	return telegram.ErrEndGroup
}

func HandleAnonVerification(cb *telegram.CallbackQuery) error {
	parts := strings.SplitN(cb.DataString(), ":", 2)
	if len(parts) != 2 {
		cb.Answer(
			"Invalid verification",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	key := parts[1]

	verify, ok := anonVerifyCache.Get(key)

	if !ok {
		cb.Answer("")
		_, err := cb.Edit(
			"⌛ <b>Session Expired</b>\n\n"+
				"<i>This verification button has expired for security reasons.</i>\n\n"+
				"💡 <b>Tip:</b> Please send the command again to generate a new verification button.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		if err != nil {
			return err
		}
		return telegram.ErrEndGroup
	}

	chatID := verify.OriginalMessage.ChannelID()
	userID := cb.SenderID

	botHasPerm, botIsAdmin, err := utils.CheckUserPermission(
		cb.Client, chatID, cb.Client.Me().ID, verify.Permission,
	)
	if err != nil {
		cb.Answer(
			"Failed to verify bot permissions.",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	if !botIsAdmin {
		anonVerifyCache.Delete(key)

		cb.Answer("I am not an admin!", &telegram.CallbackOptions{Alert: true})
		_, err := cb.Edit(fmt.Sprintf(
			"🤖 <b>I am not an Admin!</b>\n\n"+
				"<i>I need administrator rights to %s.</i>\n\n"+
				"💡 <b>Tip:</b> Please promote me to Admin and ensure I have the <i>Delete Messages</i> permission.",
			utils.PrettifyPermission(verify.Permission),
		), &telegram.SendOptions{ParseMode: "HTML"})
		if err != nil {
			return err
		}
		return telegram.ErrEndGroup
	}

	userHasPerm, userIsAdmin, err := utils.CheckUserPermission(
		cb.Client, chatID, userID, verify.Permission,
	)
	if err != nil {
		cb.Answer(
			"Failed to verify your permissions.",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	if !userIsAdmin {
		cb.Answer(
			"Who dis non-admin telling me what to do?",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	if !userHasPerm {
		cb.Answer(
			fmt.Sprintf(
				"You don't have permission to %s.",
				utils.PrettifyPermission(verify.Permission),
			),
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	if !botHasPerm {
		cb.Answer("")
		_, err := cb.Edit(
			fmt.Sprintf(
				"🚫 <b>Missing Bot Permission</b>\n\n"+
					"<i>I don't have the <code>%s</code> permission.</i>\n\n"+
					"💡 <b>Tip:</b> Please grant me this permission so I can assist you.",
				utils.PrettifyPermission(verify.Permission),
			),
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		if err != nil {
			return err
		}
		anonVerifyCache.Delete(key)
		return telegram.ErrEndGroup
	}

	anonVerifyCache.Delete(key)

	cb.Answer("✓ Verified")
	cb.Delete()

	verifiedMsg := createVerifiedMessage(verify.OriginalMessage, cb.Sender)
	return verify.Handler(verifiedMsg)
}

func createVerifiedMessage(
	original *telegram.NewMessage,
	verifiedUser *telegram.UserObj,
) *telegram.NewMessage {
	verifiedMessage := *original
	verifiedMessage.Sender = verifiedUser

	msgCopy := *verifiedMessage.Message
	msgCopy.FromID = &telegram.PeerUser{UserID: verifiedUser.ID}
	verifiedMessage.Message = &msgCopy

	return &verifiedMessage
}
