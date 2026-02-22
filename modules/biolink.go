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

var bioCache = utils.NewCache[string, any](60 * time.Minute)

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
		},
	})
}

func ShouldDeleteMsg(text string) bool {
	pattern := `\b(?:https?://|www\.)\S+|\b[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(?:/\S*)?|\B@\w{5,32}\b|\b\w{5,32}\.t\.me\b`
	re := regexp.MustCompile(pattern)
	return re.MatchString(text)
}

func setBioMode(m *telegram.NewMessage) error {
	var err error
	if _, err = m.Delete(); err != nil {
		return err
	}

	args := strings.Fields(m.Args())

	if len(args) < 1 {
		isEn, err := database.GetBioMode(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"📚 <b>Usage:</b>\n• <code>/biolink on</code> — Enable BioMode\n• <code>/biolink off</code> — Disable BioMode",
			)
			return err
		} else {
			status := "Disabled ❌"
			if isEn {
				status = "Enabled ✅"
			}
			_, err = m.Reply("🛡️ <b>BioLink Mode Status:</b> " + status + "\n\n<i>When enabled, messages from users with links in their bios will be removed.</i>")
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❗ <b>Invalid option.</b>\n\nPlease use <code>/biolink on</code> or <code>/biolink off</code>.",
		)
		return err
	}

	var msg string
	if enable {
		err := database.SetBioMode(m.ChatID())
		if err != nil {
			msg = "⚠️ <b>Oops! Failed to update BioMode.</b>\n\nSomething went wrong while trying to enable the protection. Please try again in a few moments."
		} else {
			msg = "✅ <b>BioMode enabled successfully!</b>\n\n🔍 I will now monitor bios for any links and automatically delete messages if found. Use <code>/biolink off</code> to disable this."
		}
	} else {
		err := database.DelBioMode(m.ChatID())
		if err != nil {
			msg = "⚠️ <b>Oops! Failed to update BioMode.</b>\n\nSomething went wrong while trying to disable the protection. Please try again in a few moments."
		} else {
			msg = "🛑 <b>BioMode disabled successfully!</b>\n\n🔓 I'm no longer monitoring user bios for links. Use <code>/biolink on</code> if you want to enable it again."
		}
	}

	_, err = m.Reply(msg)
	return err
}

func BioLinkHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	mode, err := database.GetBioMode(m.ChatID())
	if err != nil {
		return nil
	}
	if !mode {
		return nil
	}

	isAdmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isAdmin {
		return nil
	}

	if _, ok := m.Message.FromID.(*telegram.PeerUser); !ok {
		return nil
	}

	var bio string
	cacheKey := fmt.Sprintf("userfull_%d", m.SenderID())

	if val, ok := utils.LoadTyped[*telegram.UserFull](bioCache, cacheKey); ok {
		bio = val.About
	} else {
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
				bio = resp.FullUser.About
				bioCache.Set(cacheKey, resp.FullUser)
			} else {
				return nil
			}
		} else {
			bio = resp.FullUser.About
			bioCache.Set(cacheKey, resp.FullUser)
		}
	}

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
