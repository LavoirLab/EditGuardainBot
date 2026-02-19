package modules

import (
	"fmt"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "purge",
		ModName: "🗑️ Purge",
		Help: "🗑️ <b>Purge Messages</b> deletes multiple messages at once.\n\n" +
			"<b>Usage:</b>\n" +
			"• Reply to a message with <code>/purge</code>\n" +
			"• All messages from that message to the current one will be deleted\n\n" +
			"<b>Example:</b>\n" +
			"1. Find the first message you want to delete\n" +
			"2. Reply to it with <code>/purge</code>\n" +
			"3. All messages after that (including the replied one) will be deleted\n\n" +
			"<b>⚠️ Warning:</b> This action cannot be undone!\n" +
			"👮 Only group admins can use this command.",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:purge",
				Handler: RequirePermission("delete_messages", PurgeCmd),
			},
		},
	})
}

func PurgeCmd(m *telegram.NewMessage) error {
	if !m.IsReply() {
		if _, err := m.Delete(); err != nil {
		return err
	}
		_, err := m.Reply("⚠️ Please reply to a message to start purging from that point.")
		return err
	}

	startMsgID := m.ReplyID()
	endMsgID := m.ID

	if startMsgID >= endMsgID {
		if _, err := m.Delete(); err != nil {
		return err
	}
		_, err := m.Reply("❌ Invalid message range. Reply to an older message.")
		return err
	}

	// Calculate how many messages to delete
	messageCount := endMsgID - startMsgID + 1

	// Delete messages in batches
	var messagesToDelete []int32
	for i := startMsgID; i <= endMsgID; i++ {
		messagesToDelete = append(messagesToDelete, int32(i))
	}

	// Send notification before deleting
	notification, err := m.Reply(fmt.Sprintf("🗑️ Purging %d messages...", messageCount))
	if err != nil {
		return err
	}

	// Delete messages in batches of 50
	batchSize := 50
	deletedCount := 0

	for i := 0; i < len(messagesToDelete); i += batchSize {
		end := i + batchSize
		if end > len(messagesToDelete) {
			end = len(messagesToDelete)
		}

		batch := messagesToDelete[i:end]
		_, err := m.Client.DeleteMessages(m.ChatID(), batch)
		if err != nil {
			continue
		}
		deletedCount += len(batch)
	}

	// Update notification with results
	_, _ = notification.Edit(fmt.Sprintf("✅ Successfully purged %d messages!", deletedCount))

	// Delete the notification after 5 seconds
	go func(n *telegram.NewMessage) {
		time.Sleep(5 * time.Second)
		n.Delete()
	}(notification)

	return nil
}
