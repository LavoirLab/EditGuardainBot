package modules

import (
	"fmt"
	"log"
	"strings"
	"sync"
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
			return telegram.EndGroup
		}

		if m.IsPrivate() {
			_, err := m.Reply("This command is made to be used in group chats, not in PM!", telegram.SendOptions{})
			if err != nil {
				return err
			}
			return telegram.EndGroup
		}

		if m.Channel == nil {
			return nil
		}

		// Anonymous admin check (inline)
		if m.SenderID() == m.ChatID() || m.SenderID() == 0 {
			return handleAnonAdmin(m, permission, next)
		}

		if m.Sender == nil {
			return telegram.EndGroup
		}

		chatID := m.ChannelID()
		userID := m.SenderID()

		userHasPerm, userIsAdmin, err := utils.CheckUserPermission(
			m.Client, chatID, userID, permission,
		)
		if err != nil {
			log.Printf("Failed to check user permission: %v", err)
			_, err = m.Reply("Failed to verify your permissions.", telegram.SendOptions{})
			return err
		}

		if !userIsAdmin {
			_, err = m.Reply("Who dis non-admin telling me what to do?", telegram.SendOptions{})
			if err != nil {
				return err
			}
			return telegram.EndGroup
		}

		if !userHasPerm {
			_, err = m.Reply(
				fmt.Sprintf(
					"You don't have permission to %s.",
					utils.PrettifyPermission(permission),
				),
				telegram.SendOptions{},
			)
			if err != nil {
				return err
			}
			return telegram.EndGroup
		}

		botHasPerm, botIsAdmin, err := utils.CheckUserPermission(
			m.Client, chatID, m.Client.Me().ID, permission,
		)
		if err != nil {
			m.Client.Log.Error(fmt.Sprintf("Failed to check bot permission: %v", err))
			_, err = m.Reply("Failed to verify my permissions.", telegram.SendOptions{})
			return err
		}

		if !botIsAdmin {
			_, err = m.Reply(
				fmt.Sprintf(
					"I'm not an admin, how do you expect me to %s?",
					utils.PrettifyPermission(permission),
				),
				telegram.SendOptions{},
			)
			if err != nil {
				return err
			}
			return telegram.EndGroup
		}

		if !botHasPerm {
			_, err = m.Reply(
				fmt.Sprintf(
					"I don't have permission to %s.",
					utils.PrettifyPermission(permission),
				),
				telegram.SendOptions{},
			)
			if err != nil {
				return err
			}
			return telegram.EndGroup
		}
		return next(m)
	}
}

var (
	anonVerifyCache   = make(map[string]*AnonVerification)
	anonVerifyCacheMu sync.RWMutex
)

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
		"You are an anonymous admin. Click the button below to verify your identity.",
		telegram.SendOptions{ReplyMarkup: keyboard},
	)
	if err != nil {
		log.Printf("Failed to send anon verification message: %v", err)
		return err
	}

	anonVerifyCacheMu.Lock()
	anonVerifyCache[verifyKey] = &AnonVerification{
		OriginalMessage: m,
		Permission:      permission,
		Handler:         handler,
		CreatedAt:       time.Now(),
	}
	anonVerifyCacheMu.Unlock()

	go func() {
		time.Sleep(5 * time.Minute)
		anonVerifyCacheMu.Lock()
		delete(anonVerifyCache, verifyKey)
		anonVerifyCacheMu.Unlock()
	}()

	return telegram.EndGroup
}

func HandleAnonVerification(cb *telegram.CallbackQuery) error {
	parts := strings.SplitN(cb.DataString(), ":", 2)
	if len(parts) != 2 {
		cb.Answer(
			"Invalid verification",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.EndGroup
	}

	key := parts[1]

	anonVerifyCacheMu.RLock()
	verify, ok := anonVerifyCache[key]
	anonVerifyCacheMu.RUnlock()

	if !ok {
		cb.Answer("", nil)
		_, err := cb.Edit("This button has expired. Please use the command again.", nil)
		if err != nil {
			return err
		}
		return telegram.EndGroup
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
		return telegram.EndGroup
	}

	if !botIsAdmin {
		anonVerifyCacheMu.Lock()
		delete(anonVerifyCache, key)
		anonVerifyCacheMu.Unlock()

		cb.Answer("I'm not an admin!", &telegram.CallbackOptions{Alert: true})
		_, err := cb.Edit(fmt.Sprintf(
			"I'm not an admin, how do you expect me to %s?",
			utils.PrettifyPermission(verify.Permission),
		), nil)
		if err != nil {
			return err
		}
		return telegram.EndGroup
	}

	userHasPerm, userIsAdmin, err := utils.CheckUserPermission(
		cb.Client, chatID, userID, verify.Permission,
	)
	if err != nil {
		cb.Answer(
			"Failed to verify your permissions.",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.EndGroup
	}

	if !userIsAdmin {
		cb.Answer(
			"Who dis non-admin telling me what to do?",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.EndGroup
	}

	if !userHasPerm {
		cb.Answer(
			fmt.Sprintf(
				"You don't have permission to %s.",
				utils.PrettifyPermission(verify.Permission),
			),
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.EndGroup
	}

	if !botHasPerm {
		cb.Answer("", nil)
		_, err := cb.Edit(
			fmt.Sprintf(
				"I don't have permission to %s.",
				utils.PrettifyPermission(verify.Permission),
			),
			nil,
		)
		if err != nil {
			return err
		}
		anonVerifyCacheMu.Lock()
		delete(anonVerifyCache, key)
		anonVerifyCacheMu.Unlock()
		return telegram.EndGroup
	}

	anonVerifyCacheMu.Lock()
	delete(anonVerifyCache, key)
	anonVerifyCacheMu.Unlock()

	cb.Answer("✓ Verified", nil)
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
