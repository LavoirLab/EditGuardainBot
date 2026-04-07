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
	startID, endID, err := validatePurgeRange(m)
	if err != nil {
		return err
	}

	notification, err := m.Reply(
		fmt.Sprintf(
			"⏳ <b>Purging %d messages...</b>\n\n<i>Please wait while I clean up the chat.</i>",
			endID-startID+1,
		),
	)
	if err != nil {
		return err
	}

	deletedCount, err := runPurgeLoop(m, startID, endID, notification)
	if err != nil {
		return err
	}

	_, _ = notification.Edit(
		fmt.Sprintf(
			"✅ <b>Purge Complete!</b>\n\nSuccessfully removed <b>%d</b> messages. This message will auto-delete in 5 seconds.",
			deletedCount,
		),
	)

	ScheduleMessageDeletion(m.Client, m.ChatID(), int32(notification.ID), 5*time.Second)
	return nil
}

func validatePurgeRange(m *telegram.NewMessage) (int32, int32, error) {
	if !m.IsReply() {
		if _, err := m.Delete(); err != nil {
			return 0, 0, err
		}
		_, err := m.Reply(
			"⚠️ <b>Purge requires a reply!</b>\n\n"+
				"<i>Please reply to the message where you want the purge to <b>start</b>.</i>\n\n"+
				"💡 <b>Tip:</b> All messages between the replied message and your command will be deleted.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		if err != nil {
			return 0, 0, err
		}
		return 0, 0, telegram.ErrEndGroup
	}

	startMsgID := int32(m.ReplyID())
	endMsgID := int32(m.ID)

	if startMsgID >= endMsgID {
		if _, err := m.Delete(); err != nil {
			return 0, 0, err
		}
		_, err := m.Reply(
			"❌ <b>Invalid Message Range</b>\n\n"+
				"<i>The replied message must be older than the current command.</i>\n\n"+
				"💡 <b>Tip:</b> Scroll up to find the first message you want to delete and reply to it.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		if err != nil {
			return 0, 0, err
		}
		return 0, 0, telegram.ErrEndGroup
	}

	return startMsgID, endMsgID, nil
}

func runPurgeLoop(m *telegram.NewMessage, startID, endID int32, notification *telegram.NewMessage) (int, error) {
	var messagesToDelete []int32
	for i := startID; i <= endID; i++ {
		messagesToDelete = append(messagesToDelete, i)
	}

	batchSize := 100
	deletedCount := 0
	totalToPurge := len(messagesToDelete)

	for i := 0; i < totalToPurge; i += batchSize {
		end := i + batchSize
		if end > totalToPurge {
			end = totalToPurge
		}

		batch := messagesToDelete[i:end]
		if _, err := m.Client.DeleteMessages(m.ChatID(), batch); err == nil {
			deletedCount += len(batch)
		}

		if totalToPurge > 200 && deletedCount%200 == 0 {
			_, _ = notification.Edit(
				fmt.Sprintf(
					"⏳ <b>Purging messages...</b>\n\n<i>Progress: %d / %d deleted.</i>",
					deletedCount,
					totalToPurge,
				),
			)
		}
	}

	return deletedCount, nil
}
