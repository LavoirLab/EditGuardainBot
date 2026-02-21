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
		Help: "🛡️ <b>BioMode</b> monitors user bios and deletes messages if they contain URLs.\n\n" +
			"<b>Usage:</b>\n" +
			"➤ <code>/biolink on</code> - Enable BioMode\n" +
			"➤ <code>/biolink off</code> - Disable BioMode\n\n" +
			"🚫 When enabled, users with links in their bios won't be able to send messages.\n" +
			"👮 Only admins can enable or disable this feature.",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:biolink",
				Handler: RequirePermission("change_info", setBioMode),
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
		if isEn, err := database.GetBioMode(m.ChatID()); err != nil {
			_, err = m.Reply("📚 Usage: <code>/biolink on</code> | <code>/biolink off</code>")
			return err
		} else {
			_, err = m.Reply("Currently bioLink mode is " + map[bool]string{true: "Enabled", false: "Disabled"}[isEn] + " for your chat")
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply("❗ Invalid option. Use <code>on</code> or <code>off</code>.")
		return err
	}

	var msg string
	if enable {
		err := database.SetBioMode(m.ChatID())
		if err != nil {
			msg = fmt.Sprintf("⚠️ <b>Oops! Failed to enable BioMode.</b>\n\n🚫 An error occurred while trying to turn it on.\n\n<b>Error:</b> <code>%v</code>\n\n🔁 Please try again later.", err)
		} else {
			msg = "✅ <b>BioMode enabled successfully!</b>\n\n🔍 I will now monitor bios for any links and automatically delete messages if found.\n\n🛡 Stay safe!"
		}
	} else {
		err := database.DelBioMode(m.ChatID())
		if err != nil {
			msg = fmt.Sprintf("⚠️ <b>Oops! Failed to disable BioMode.</b>\n\n🚫 An error occurred while trying to turn it off.\n\n<b>Error:</b> <code>%v</code>\n\n🔁 Please try again later.", err)
		} else {
			msg = "🛑 <b>BioMode disabled successfully!</b>\n\n🔓 I'm no longer monitoring user bios for links in this group.\n\n✅ You're back to normal behavior."
		}
	}

	_, err = m.Reply(msg)
	return err
}

func BioLinkHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
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
	msg := fmt.Sprintf("🛡️ <b>BioLink Protected!</b>\n\n%s, <i>your message was removed because your profile bio contains a link. To prevent spam, links in bios are not allowed here.</i> 🚫", mention)
	_, _ = m.Reply(msg)
	return telegram.ErrEndGroup
}
