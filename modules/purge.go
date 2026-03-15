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
		Help: `<emoji id='5445267414562389170'>🗑️</emoji> <b>ᴘᴜʀɢᴇ ᴍᴇssᴀɢᴇs</b> ᴅᴇʟᴇᴛᴇs ᴍᴜʟᴛɪᴘʟᴇ ᴍᴇssᴀɢᴇs ᴀᴛ ᴏɴᴄᴇ.

<b>ᴜsᴀɢᴇ:</b>
<blockquote>• ʀᴇᴘʟʏ ᴛᴏ ᴀ ᴍᴇssᴀɢᴇ ᴡɪᴛʜ <code>/purge</code>
• ᴀʟʟ ᴍᴇssᴀɢᴇs ғʀᴏᴍ ᴛʜᴀᴛ ᴍᴇssᴀɢᴇ ᴛᴏ ᴛʜᴇ ᴄᴜʀʀᴇɴᴛ ᴏɴᴇ ᴡɪʟʟ ʙᴇ ᴅᴇʟᴇᴛᴇᴅ</blockquote>

<b>ᴇxᴀᴍᴘʟᴇ:</b>
<blockquote>1. ғɪɴᴅ ᴛʜᴇ ғɪʀsᴛ ᴍᴇssᴀɢᴇ ʏᴏᴜ ᴡᴀɴᴛ ᴛᴏ ᴅᴇʟᴇᴛᴇ
2. ʀᴇᴘʟʏ ᴛᴏ ɪᴛ ᴡɪᴛʜ <code>/purge</code>
3. ᴀʟʟ ᴍᴇssᴀɢᴇs ᴀғᴛᴇʀ ᴛʜᴀᴛ (ɪɴᴄʟᴜᴅɪɴɢ ᴛʜᴇ ʀᴇᴘʟɪᴇᴅ ᴏɴᴇ) ᴡɪʟʟ ʙᴇ ᴅᴇʟᴇᴛᴇᴅ</blockquote>

<b><emoji id='5447644880824181073'>⚠️</emoji> ᴡᴀʀɴɪɴɢ:</b> ᴛʜɪs ᴀᴄᴛɪᴏɴ ᴄᴀɴɴᴏᴛ ʙᴇ ᴜɴᴅᴏɴᴇ!
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴜsᴇ ᴛʜɪs ᴄᴏᴍᴍᴀɴᴅ.`,
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
		_, err := m.Reply(
			"⚠️ <b>Purge requires a reply!</b>\n\n"+
				"<i>Please reply to the message where you want the purge to <b>start</b>.</i>\n\n"+
				"💡 <b>Tip:</b> All messages between the replied message and your command will be deleted.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return err
	}

	startMsgID := m.ReplyID()
	endMsgID := m.ID

	if startMsgID >= endMsgID {
		if _, err := m.Delete(); err != nil {
			return err
		}
		_, err := m.Reply(
			"❌ <b>Invalid Message Range</b>\n\n"+
				"<i>The replied message must be older than the current command.</i>\n\n"+
				"💡 <b>Tip:</b> Scroll up to find the first message you want to delete and reply to it.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
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
	notification, err := m.Reply(
		fmt.Sprintf(
			"⏳ <b>Purging %d messages...</b>\n\n<i>Please wait while I clean up the chat.</i>",
			messageCount,
		),
	)
	if err != nil {
		return err
	}

	// Delete messages in batches of 50
	batchSize := 100
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

		// Update progress for large purges
		if len(messagesToDelete) > 200 && deletedCount%200 == 0 {
			_, _ = notification.Edit(
				fmt.Sprintf(
					"⏳ <b>Purging messages...</b>\n\n<i>Progress: %d / %d deleted.</i>",
					deletedCount,
					messageCount,
				),
			)
		}
	}

	// Update notification with results
	_, _ = notification.Edit(
		fmt.Sprintf(
			"✅ <b>Purge Complete!</b>\n\nSuccessfully removed <b>%d</b> messages. This message will auto-delete in 5 seconds.",
			deletedCount,
		),
	)

	// Delete the notification after 5 seconds
	ScheduleMessageDeletion(
		m.Client,
		m.ChatID(),
		int32(notification.ID),
		5*time.Second,
	)

	return nil
}
