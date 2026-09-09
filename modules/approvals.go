package modules

import (
	"fmt"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "approvals",
		ModName: "✅ Approvals",
		Help: `<emoji id='5443232278963982555'>✅</emoji> <b>Usᴇʀ Aᴘᴘʀᴏᴠᴀʟs</b>

<i>ᴀᴘᴘʀᴏᴠᴇs ᴛʀᴜsᴛᴇᴅ ᴜsᴇʀs sᴏ ᴛʜᴇʏ ᴀʀᴇ ᴇxᴇᴍᴘᴛ ғʀᴏᴍ ᴀʟʟ ᴀᴜᴛᴏᴍᴀᴛᴇᴅ ᴍᴏᴅᴇʀᴀᴛɪᴏɴ ᴀᴄᴛɪᴏɴs ʟɪᴋᴇ ʟɪɴᴋ, ᴘʀᴏᴍᴏ, ʙɪᴏʟɪɴᴋ ᴀɴᴅ ʟᴀɴɢᴜᴀɢᴇ ғɪʟᴛᴇʀs.</i>

<b><emoji id='5341715473882955310'>🔧</emoji> ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/approve</code> — ᴀᴘᴘʀᴏᴠᴇ ᴀ ᴜsᴇʀ (ʀᴇᴘʟʏ / ᴍᴇɴᴛɪᴏɴ / ɪᴅ) <emoji id='6084551628461967966'>✅</emoji>
• <code>/unapprove</code> ᴏʀ <code>/disapprove</code> — ʀᴇᴠᴏᴋᴇ ᴀᴘᴘʀᴏᴠᴀʟ <emoji id='5465665476971471368'>🚫</emoji>
• <code>/approved</code> — ᴄʜᴇᴄᴋ ᴡʜᴇᴛʜᴇʀ ᴀ ᴜsᴇʀ ɪs ᴀᴘᴘʀᴏᴠᴇᴅ <emoji id='5447067204298713202'>🔍</emoji></blockquote>

<b><emoji id='5443038326535759644'>ℹ️</emoji> ɴᴏᴛᴇs:</b>
<blockquote>– ᴀᴘᴘʀᴏᴠᴇᴅ ᴜsᴇʀs ʙʏᴘᴀss ᴀʟʟ ᴀᴜᴛᴏᴍᴀᴛɪᴄ ᴘʀᴏᴛᴇᴄᴛɪᴏɴs.
– ᴛʜᴇʏ ᴄᴀɴ sᴛɪʟʟ ʙᴇ ᴘᴜɴɪsʜᴇᴅ ᴍᴀɴᴜᴀʟʟʏ ʙʏ ᴀᴅᴍɪɴs.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴀᴘᴘʀᴏᴠᴇ ᴏʀ ʀᴇᴠᴏᴋᴇ ᴀᴘᴘʀᴏᴠᴀʟs.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:approve",
				Handler: RequirePermission("delete_messages", approveCmd),
			},
			{
				Pattern: "command:unapprove",
				Handler: RequirePermission("delete_messages", unapproveCmd),
			},
			{
				Pattern: "command:disapprove",
				Handler: RequirePermission("delete_messages", unapproveCmd),
			},
			{
				Pattern: "command:approved",
				Handler: RequirePermission("delete_messages", approvedCmd),
			},
		},
	})
}

func approveCmd(m *telegram.NewMessage) error {
	userID := utils.ExtractUser(m)
	if userID == 0 {
		return telegram.ErrEndGroup
	}
	if err := database.ApproveUser(m.ChatID(), userID); err != nil {
		_, e := m.Reply("⚠️ Failed to approve user.")
		if e != nil {
			return e
		}
		return err
	}
	_, err := m.Reply(
		fmt.Sprintf(
			"✅ User <code>%d</code> is now approved and exempt from automated actions.",
			userID,
		),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	return err
}

func unapproveCmd(m *telegram.NewMessage) error {
	userID := utils.ExtractUser(m)
	if userID == 0 {
		return telegram.ErrEndGroup
	}
	if err := database.UnapproveUser(m.ChatID(), userID); err != nil {
		_, e := m.Reply("⚠️ Failed to unapprove user.")
		if e != nil {
			return e
		}
		return err
	}
	_, err := m.Reply(
		fmt.Sprintf("🚫 User <code>%d</code> has been unapproved.", userID),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	return err
}

func approvedCmd(m *telegram.NewMessage) error {
	userID := utils.ExtractUser(m)
	if userID == 0 {
		return telegram.ErrEndGroup
	}
	approved, err := database.IsUserApproved(m.ChatID(), userID)
	if err != nil {
		_, e := m.Reply("⚠️ Failed to check approval status.")
		if e != nil {
			return e
		}
		return err
	}
	status := "not approved"
	if approved {
		status = "approved ✅"
	}
	_, err = m.Reply(
		fmt.Sprintf("👤 User <code>%d</code> is %s.", userID, status),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	return err
}
