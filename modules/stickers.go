package modules

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "stickers",
		ModName: "🖼️ Stickers",
		Help: `🖼️ <b>Sᴛɪᴄᴋᴇʀ Fɪʟᴛᴇʀ</b> ʟᴇᴛs ᴀᴅᴍɪɴs ʀᴇsᴛʀɪᴄᴛ ꜱᴘᴇᴄɪꜰɪᴄ ꜱᴛɪᴄᴋᴇʀꜱ ꜰʀᴏᴍ ʙᴇɪɴɢ ꜱᴇɴᴛ ɪɴ ᴛʜᴇ ɢʀᴏᴜᴘ.

<b>ʜᴏᴡ ɪᴛ ᴡᴏʀᴋs:</b>
<blockquote>➤ ʙᴏᴛ ꜱᴀᴠᴇꜱ ᴛʜᴇ ꜱᴛɪᴄᴋᴇʀ ɪᴅ ᴏꜰ ᴀ ʀᴇᴘʟɪᴇᴅ ꜱᴛɪᴄᴋᴇʀ
➤ ᴏɴᴄᴇ ʙʟᴏᴄᴋᴇᴅ, ᴀɴʏ ᴜꜱᴇʀ ꜱᴇɴᴅɪɴɢ ᴛʜᴀᴛ ꜱᴛɪᴄᴋᴇʀ ᴡɪʟʟ ʙᴇ ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇʟᴇᴛᴇᴅ</blockquote>

<b>ᴜsᴀɢᴇ:</b>
<blockquote>➤ <code>/blocksticker</code> – ʀᴇᴘʟʏ ᴛᴏ ᴀ ꜱᴛɪᴄᴋᴇʀ ᴛᴏ ʙʟᴏᴄᴋ ɪᴛ
➤ <code>/bs</code> – ꜱʜᴏʀᴛ ᴀʟɪᴀꜱ ꜰᴏʀ ʙʟᴏᴄᴋɪɴɢ

➤ <code>/unblocksticker</code> – ʀᴇᴘʟʏ ᴛᴏ ᴀ ʙʟᴏᴄᴋᴇᴅ ꜱᴛɪᴄᴋᴇʀ ᴛᴏ ʀᴇᴍᴏᴠᴇ ɪᴛ
➤ <code>/ubs</code> – ꜱʜᴏʀᴛ ᴀʟɪᴀꜱ ꜰᴏʀ ᴜɴʙʟᴏᴄᴋɪɴɢ

➤ <code>/listblockedstickers</code> – ꜱʜᴏᴡ ᴀʟʟ ʙʟᴏᴄᴋᴇᴅ ꜱᴛɪᴄᴋᴇʀꜱ
➤ <code>/slist</code> – ꜱʜᴏʀᴛ ᴀʟɪᴀꜱ ꜰᴏʀ ʟɪꜱᴛ</blockquote>

<blockquote><emoji id='5929345782560853059'>🚫</emoji> ʙʟᴏᴄᴋᴇᴅ ꜱᴛɪᴄᴋᴇʀꜱ ᴀʀᴇ ʀᴇᴍᴏᴠᴇᴅ ɪɴꜱᴛᴀɴᴛʟʏ ᴀꜰᴛᴇʀ ꜱᴇɴᴅɪɴɢ.
<emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ᴀᴅᴍɪɴꜱ ᴄᴀɴ ᴍᴀɴᴀɢᴇ ᴛʜᴇ ꜰɪʟᴛᴇʀ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:(blocksticker|sblock|bs)",
				Handler: RequirePermission("delete_messages", BlockStickerCmd),
			},
			{
				Pattern: "command:(unblocksticker|sunblock|ubs)",
				Handler: RequirePermission(
					"delete_messages",
					UnblockStickerCmd,
				),
			},
			{
				Pattern: "command:(listblockedstickers|slist|lbs)",
				Handler: ListBlockedStickersCmd,
			},
			{
				Pattern: "callback:^blockedlist_",
				Handler: handleBlockedListCallback,
			},
		},
	})
}

func BlockStickerCmd(m *telegram.NewMessage) error {
	reply, _ := m.GetReplyMessage()
	if reply == nil || reply.Sticker() == nil || reply.File == nil {
		_, err := m.Reply("❌ Please reply to a sticker to block it.")
		return err
	}

	fileID := reply.File.FileID
	blocked, err := database.BlockSticker(m.ChatID(), fileID)
	if err != nil {
		_, err = m.Reply(
			"⚠️ <b>Something went wrong while blocking the sticker.</b>",
		)
		return err
	}

	if !blocked {
		_, err = m.Reply("This sticker is already blocked.")
		return err
	}

	_, err = m.Reply("✅ Sticker blocked.")
	return err
}

func UnblockStickerCmd(m *telegram.NewMessage) error {
	reply, _ := m.GetReplyMessage()
	if reply == nil || reply.Sticker() == nil || reply.File == nil {
		_, err := m.Reply("❌ Please reply to a sticker to unblock it.")
		return err
	}

	fileID := reply.File.FileID
	unblocked, err := database.UnblockSticker(m.ChatID(), fileID)
	if err != nil {
		_, err = m.Reply(
			"⚠️ <b>Something went wrong while unblocking the sticker.</b>",
		)
		return err
	}

	if !unblocked {
		_, err = m.Reply("This sticker is not blocked.")
		return err
	}

	_, err = m.Reply("✅ Sticker unblocked.")
	return err
}

func ListBlockedStickersCmd(m *telegram.NewMessage) error {
	if m.IsGroup() {
		btn := telegram.NewKeyboard().AddRow(
			telegram.Button.URL("📋 View Blocked Stickers", fmt.Sprintf("https://t.me/%s?start=blockedlist_%d", m.Client.Me().Username, m.ChatID())),
		).
			Build()

		_, err := m.Reply(
			"Click the button below to view the list of blocked stickers in private.",
			&telegram.SendOptions{ReplyMarkup: btn},
		)
		return err
	}
	if m.IsPrivate() {
		args := strings.Fields(m.Args())
		if len(args) > 0 {
			chatID, err := strconv.ParseInt(args[0], 10, 64)
			if err == nil {
				return listBlockedStickers(
					m.Client,
					m.Sender.ID,
					cleanChatID(chatID),
					0,
					nil,
				)
			}
		}
		_, err := m.Reply(
			"<b>Usage:</b> <code>/slist &lt;chat_id&gt;</code> in PM, or use it in a group.",
		)
		return err
	}
	return nil
}

func StickerBlockHandler(m *telegram.NewMessage) error {
	if m.Sticker() == nil || m.File == nil || m.IsPrivate() ||
		m.IsChannelPost() {
		return nil
	}

	isAdmin, err := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if err != nil || isAdmin {
		return nil
	}

	fileID := m.File.FileID
	isBlocked, err := database.IsStickerBlocked(m.ChatID(), fileID)
	if err != nil || !isBlocked {
		return nil
	}

	_, err = m.Delete()
	if err != nil {
		return nil
	}

	mention := utils.MentionHTML(m.Sender)
	m, _ = m.Client.SendMessage(
		m.ChatID(),
		fmt.Sprintf(
			"⚠️ %s, that sticker is not allowed in this group.",
			mention,
		),
		&telegram.SendOptions{ParseMode: "HTML"},
	)

	if m != nil {
		ScheduleMessageDeletion(
			m.Client,
			m.ChatID(),
			int32(m.ID),
			20*time.Second,
		)
	}
	return telegram.ErrEndGroup
}

func handleBlockedListCallback(c *telegram.CallbackQuery) error {
	data := c.DataString()
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return nil
	}

	chatID, _ := strconv.ParseInt(parts[1], 10, 64)
	offset, _ := strconv.Atoi(parts[2])

	return listBlockedStickers(c.Client, c.Sender.ID, chatID, offset, c)
}

func listBlockedStickers(
	client *telegram.Client,
	userID, chatID int64,
	offset int,
	c *telegram.CallbackQuery,
) error {
	stickers, err := database.GetBlockedStickers(chatID)
	if err != nil {
		if c != nil {
			c.Answer("Error fetching stickers.")
		}
		return nil
	}

	if len(stickers) == 0 {
		if c != nil {
			c.Edit("No stickers are blocked in this group.")
		} else {
			client.SendMessage(userID, "No stickers are blocked in this group.")
		}
		return nil
	}

	if c != nil {
		c.Answer("")
		c.Delete()
	}

	end := offset + 10
	if end > len(stickers) {
		end = len(stickers)
	}

	for i := offset; i < end; i++ {

		media, err := telegram.ResolveBotFileID(stickers[i])
		if err != nil {
			log.Printf("Failed to resolve sticker %s: %v", stickers[i], err)
		} else {

			umm, err := client.SendMedia(userID, media)
			if err != nil {
				log.Printf("Failed to send sticker %s: %v", stickers[i], err)
			} else {
				ScheduleMessageDeletion(
					client,
					userID,
					int32(umm.ID),
					15*time.Second,
				)
			}
		}

		time.Sleep(time.Second)
	}

	if end < len(stickers) {
		btn := telegram.NewKeyboard().AddRow(
			telegram.Button.Data("Load next 10 ➡️", fmt.Sprintf("blockedlist_%d_%d", chatID, end)),
		).
			Build()
		client.SendMessage(
			userID,
			fmt.Sprintf("Listed %d/%d stickers.", end, len(stickers)),
			&telegram.SendOptions{ReplyMarkup: btn},
		)
	} else {
		client.SendMessage(userID, fmt.Sprintf("✅ All %d blocked stickers have been listed.", len(stickers)))
	}

	return nil
}

// Exported for start.go
func HandleBlockedListDeepLink(m *telegram.NewMessage, chatID int64) error {
	return listBlockedStickers(m.Client, m.Sender.ID, chatID, 0, nil)
}

func cleanChatID(input int64) int64 {
	// supergroup / channel  (-100xxxxxxxxxx)
	if input <= -1000000000000 {
		return -(input + 1000000000000)
	}

	// old basic group (-xxxxxxxxx)
	if input < 0 {
		return -input
	}

	return input
}
