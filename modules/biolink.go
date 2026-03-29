package modules

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func getBioLinkDashboard(enabled bool) (string, telegram.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("🛡️ <b>BioLink Protection</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n\n"+
		"<blockquote expandable>"+
		"When enabled, I monitor user bios for any URLs or links. If a user tries to send a message while having a link in their bio, the message will be removed immediately.\n\n"+
		"<b>Action:</b> Messages from flagged users are deleted to prevent potential spam or promotion."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> This is an effective way to stop 'bio-spammers' before they even start.", status)

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data(toggleLabel, "biolink_toggle:"+toggleVal),
		telegram.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard
}

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "biolink",
		ModName: "🛡️ BioMode",
		Help: `<emoji id='5465154440287757794'>🛡️</emoji> <b>ʙɪᴏᴍᴏᴅᴇ</b> ᴍᴏɴɪᴛᴏʀs ᴜsᴇʀ ʙɪᴏs ᴀɴᴅ ᴅᴇʟᴇᴛᴇs ᴍᴇssᴀɢᴇs ɪғ ᴛʜᴇʏ ᴄᴏɴᴛᴀɪɴ ᴜʀʟs.

<b>ᴜsᴀɢᴇ:</b>
➤ <code>/biolink on</code> - ᴇɴᴀʙʟᴇ ʙɪᴏᴍᴏᴅᴇ
➤ <code>/biolink off</code> - ᴅɪsᴀʙʟᴇ ʙɪᴏᴍᴏᴅᴇ

<blockquote><emoji id='5929345782560853059'>🚫</emoji> ᴡʜᴇɴ ᴇɴᴀʙʟᴇᴅ, ᴜsᴇʀs ᴡɪᴛʜ ʟɪɴᴋs ɪɴ ᴛʜᴇɪʀ ʙɪᴏs ᴡᴏɴ'ᴛ ʙᴇ ᴀʙʟᴇ ᴛᴏ sᴇɴᴅ ᴍᴇssᴀɢᴇs.
<emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ᴀᴅᴍɪɴs ᴄᴀɴ ᴇɴᴀʙʟᴇ ᴏʀ ᴅɪsᴀʙʟᴇ ᴛʜɪs ғᴇᴀᴛᴜʀᴇ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:biolink",
				Handler: RequirePermission("delete_messages", setBioMode),
			},
			{
				Pattern: "callback:^biolink_toggle:(on|off)$",
				Handler: handleBioLinkToggle,
			},
		},
	})
}

func handleBioLinkToggle(c *telegram.CallbackQuery) error {
	if isAdmin, _ := utils.IsAdmin(c.Client, c.ChatID, c.SenderID); !isAdmin {
		c.Answer(
			"👤 Admin Access Required",
			&telegram.CallbackOptions{Alert: true},
		)
		return telegram.ErrEndGroup
	}

	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	var err error
	if enable {
		err = database.SetBioMode(c.ChatID)
	} else {
		err = database.DelBioMode(c.ChatID)
	}

	if err != nil {
		c.Answer("⚠️ Update failed.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	text, keyboard := getBioLinkDashboard(enable)
	c.Answer("✓ Settings Updated")
	_, err = c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

var bioLinkRegex = regexp.MustCompile(
	`\b(?:https?://|www\.)\S+|\b[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(?:/\S*)?|\B@\w{5,32}\b|\b\w{5,32}\.t\.me\b`,
)

func ShouldDeleteMsg(text string) bool {
	return bioLinkRegex.MatchString(text)
}

func setBioMode(m *telegram.NewMessage) error {
	var err error
	if _, err = m.Delete(); err != nil {
		return err
	}

	args := strings.Fields(m.Args())

	if len(args) < 1 {
		isEn, _ := database.GetBioMode(m.ChatID())
		text, keyboard := getBioLinkDashboard(isEn)
		_, err = m.Reply(text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❗ <b>Invalid option.</b>\n\nPlease use <code>/biolink on</code> or <code>/biolink off</code>.",
		)
		return err
	}

	if enable {
		err = database.SetBioMode(m.ChatID())
	} else {
		err = database.DelBioMode(m.ChatID())
	}

	if err != nil {
		_, err = m.Reply(
			"⚠️ <b>Oops! Failed to update BioMode.</b>\n\nSomething went wrong while trying to update the protection. Please try again in a few moments.",
		)
		return err
	}

	text, keyboard := getBioLinkDashboard(enable)
	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func BioLinkHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	mode, err := database.GetBioMode(m.ChatID())
	if err != nil || !mode {
		return nil
	}

	isAdmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isAdmin {
		return nil
	}

	if _, ok := m.Message.FromID.(*telegram.PeerUser); !ok ||
		m.SenderID() == 0 {
		return nil
	}

	resp, err := m.Client.UsersGetFullUser(&telegram.InputUserObj{
		UserID:     m.Sender.ID,
		AccessHash: m.Sender.AccessHash,
	})
	if err != nil {
		if wait := telegram.GetFloodWait(err); wait > 0 && wait < 15 {
			time.Sleep(time.Duration(wait) * time.Second)
			resp, err = m.Client.UsersGetFullUser(&telegram.InputUserObj{
				UserID:     m.Sender.ID,
				AccessHash: m.Sender.AccessHash,
			})
			if err != nil {
				return nil
			}
		} else {
			return nil
		}
	}

	bio := resp.FullUser.About
	if bio == "" || !ShouldDeleteMsg(bio) {
		return nil
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	mention := utils.MentionHTML(m.Sender)
	msg := fmt.Sprintf(
		"🛡️ <b>BioLink Protected!</b>\n\n%s, <i>your message was removed because your profile bio contains a link.</i>\n\n💡 <b>Tip:</b> Remove the link from your bio to be able to send messages in this group.",
		mention,
	)
	_, _ = m.Reply(msg)
	return telegram.ErrEndGroup
}
