package modules

import (
	"errors"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
)

func OnMessageFnc(m *telegram.NewMessage) error {
	go func() {
		if m.IsPrivate() {
			database.AddServedUser(m.ChatID())
		} else {
			database.AddServedChat(m.ChatID())
		}
	}()

	if _, ok := commandSet[m.GetCommand()]; ok {
		return nil
	}

	// All message handlers
	handlers := []func(*telegram.NewMessage) error{
		// Existing handlers
		handleHashtags,         // Hashtag blocking
		handlePromoMessages,    // Promo message blocking
		handleForwardedMessage, // Forward blocking

		deleteLongMessage,
		deleteLinkMessage,
		DeleteAbuseHandle,
		handlePhoneNumber,   // Phone number blocking
		handleMediaDelete,   // Media auto-delete
		handleMsgAutoDelete, // Message auto-delete

		// NEW HANDLERS
		deleteUserMsgIfBio,
	}

	for _, handler := range handlers {
		if err := handler(m); err != nil {
			if errors.Is(err, telegram.EndGroup) || telegram.MatchError(err, "You can't delete one of the messages you tried to delete, most likely because it is a service message") {
				return telegram.EndGroup
			}
			return L(m, "Modules -> message -> Handler", err)
		}
	}

	return nil
}
