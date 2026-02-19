package modules

import (
	"fmt"
	"log"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "watcher",
		ModName: "Watcher",
		Help:    "",
		Handlers: []HandlerConfig{
			{
				Pattern: "action",
				Handler: handleChatAction,
				Group:   10,
			},
			{
				Pattern: "participant",
				Handler: handleParticipantUpdate,
				Group:   10,
			},
		},
	})
}

func handleParticipantUpdate(pu *telegram.ParticipantUpdate) error {
	if pu == nil || pu.Channel == nil {
		return nil
	}

	self := pu.Client.Me()
	chatID := pu.ChannelID()
	userID := pu.UserID()
	oldStatus := getParticipantStatus(pu.Old)
	newStatus := getParticipantStatus(pu.New)

	// bot itself

	if userID == self.ID {
		log.Printf(
			"MyChatMember | chat_id=%d %s -> %s",
			chatID,
			oldStatus,
			newStatus,
		)

		// bot promoted to admin
		if newStatus == "administrator" {
			utils.UpdateAdminFromParticipant(
				pu.Client,
				chatID,
				self.ID,
				pu.New,
			)
			return telegram.ErrEndGroup
		}

		// bot demoted / removed
		if newStatus == "member" ||
			newStatus == "left" ||
			newStatus == "kicked" {
			utils.ClearChatAdminCache(chatID)
			return telegram.ErrEndGroup
		}

		return telegram.ErrEndGroup
	}

	// ------------------------------------------------
	// CHAT_MEMBER  (any other user)
	// ------------------------------------------------
	if pu.User == nil {
		return nil
	}

	log.Printf(
		"ChatMember | user=%d chat=%d %s -> %s",
		userID,
		chatID,
		oldStatus,
		newStatus,
	)

	// promoted / rights changed
	if newStatus == "administrator" || newStatus == "creator" {
		utils.UpdateAdminFromParticipant(
			pu.Client,
			chatID,
			userID,
			pu.New,
		)
		return telegram.ErrEndGroup
	}

	// demoted / left / kicked / restricted
	if newStatus == "member" ||
		newStatus == "restricted" ||
		newStatus == "left" ||
		newStatus == "kicked" {
		utils.RemoveAdmin(chatID, userID)
		return telegram.ErrEndGroup
	}

	return telegram.ErrEndGroup
}

func getParticipantStatus(p telegram.ChannelParticipant) string {
	if p == nil {
		return "left"
	}

	switch p.(type) {
	case *telegram.ChannelParticipantCreator:
		return "creator"
	case *telegram.ChannelParticipantAdmin:
		return "administrator"
	case *telegram.ChannelParticipantSelf,
		*telegram.ChannelParticipantObj:
		return "member"
	case *telegram.ChannelParticipantLeft:
		return "left"
	case *telegram.ChannelParticipantBanned:
		return "kicked"
	default:
		return "unknown"
	}
}

func handleChatAction(m *telegram.NewMessage) error {
	if !isValidChatType(m) {
		warnAndLeave(m.Client, m.ChannelID())
		return telegram.ErrEndGroup
	}

	botID := m.Client.Me().ID
	switch action := m.Action.(type) {
	case *telegram.MessageActionChatAddUser:
		// Check if bot was added
		for _, userID := range action.Users {
			if userID == botID {
				return handleBotAdded(m)
			}
		}

	case *telegram.MessageActionChatDeleteUser:
		// Check if bot was removed
		if action.UserID == botID {
			return handleBotRemoved(m)
		}
	}

	return nil
}

func handleBotAdded(m *telegram.NewMessage) error {
	chatType := m.ChatType()

	switch chatType {
	case telegram.EntityChannel:
		return nil

	default:
		// SuperGroup - proceed normally
		welcomeText := fmt.Sprintf(`👋 <b>Hello!</b>

Thanks for adding me to <b>%s</b>! I'm %s, your group management assistant.

Use /help to see what I can do!`, getTitle(m), m.Client.Me().FirstName)

		_, err := m.Reply(welcomeText, &telegram.SendOptions{ParseMode: "HTML"})
		return err
	}
}

func handleBotRemoved(m *telegram.NewMessage) error {
	return nil
}

func getTitle(m *telegram.NewMessage) string {
	if m.Channel != nil {
		return m.Channel.Title
	}
	return "this group"
}

func isValidChatType(m *telegram.NewMessage) bool {
	return m.ChatType() != telegram.EntityChat ||
		(m.Channel != nil && m.Channel.Megagroup)
}

func warnAndLeave(client *telegram.Client, chatID int64) {
	text := `⚠️ <b>Basic Groups Not Supported</b>

I only work in SuperGroups! Please convert this group to a SuperGroup first.

<b>How to convert:</b>
1. Go to Group Settings
2. Select "Chat History"
3. Make it "Visible"

After converting, add me again and I'll work perfectly! 😊`

	client.SendMessage(chatID, text)

	go func() {
		time.Sleep(1 * time.Second)
		client.LeaveChannel(chatID)
	}()
	return
}
