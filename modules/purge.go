package modules

import (
	"fmt"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func init() {
	AddHelp(
		"🗑️ Purge",
		"purge_help",
		"🗑️ <b>Purge Messages</b> deletes multiple messages at once.\n\n"+
			"<b>Usage:</b>\n"+
			"• Reply to a message with <code>/purge</code>\n"+
			"• All messages from that message to the current one will be deleted\n\n"+
			"<b>Example:</b>\n"+
			"1. Find the first message you want to delete\n"+
			"2. Reply to it with <code>/purge</code>\n"+
			"3. All messages after that (including the replied one) will be deleted\n\n"+
			"<b>⚠️ Warning:</b> This action cannot be undone!\n"+
			"👮 Only group admins can use this command.",
	)
}

func PurgeCmd(m *telegram.NewMessage) error {
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}

	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> purge -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Delete()
		m.Respond("Access denied: Only group admins can use this command.")
		return telegram.EndGroup
	}

	if !m.IsReply() {
		m.Delete()
		m.Respond("⚠️ Please reply to a message to start purging from that point.")
		return telegram.EndGroup
	}

	startMsgID := m.ReplyID()
	endMsgID := m.ID

	if startMsgID >= endMsgID {
		m.Delete()
		m.Respond("❌ Invalid message range. Reply to an older message.")
		return telegram.EndGroup
	}

	// Calculate how many messages to delete
	messageCount := endMsgID - startMsgID + 1

	// Delete messages in batches (Telegram allows max 100 messages per request)
	var messagesToDelete []int32
	for i := startMsgID; i <= endMsgID; i++ {
		messagesToDelete = append(messagesToDelete, int32(i))
	}

	// Send notification before deleting
	notification, err := m.Respond(fmt.Sprintf("🗑️ Purging %d messages...", messageCount))
	if err != nil {
		return L(m, "Modules -> purge -> Respond", err)
	}

	// Delete messages in batches of 100
	batchSize := 50
	deletedCount := 0

	for i := 0; i < len(messagesToDelete); i += batchSize {
		end := i + batchSize
		if end > len(messagesToDelete) {
			end = len(messagesToDelete)
		}

		batch := messagesToDelete[i:end]
		affectedMessages, err := m.Client.DeleteMessages(m.ChatID(), batch)
		if err != nil {
			L(m, "Modules -> purge -> DeleteMessages", err)
			continue
		}
		deletedCount += int(affectedMessages.PtsCount)
	}

	// Update notification with results
	notification.Edit(fmt.Sprintf("✅ Successfully purged %d messages!", deletedCount))

	// Delete the notification after 5 seconds
	go func(m *telegram.NewMessage) {
		time.Sleep(5 * time.Second)
		m.Delete()
	}(notification)

	return telegram.EndGroup
}
