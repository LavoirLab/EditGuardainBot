package modules

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
	"main/utils/cache"
	"main/database"
)

var bioCache = cache.NewCache[string, any](60 * time.Minute)

func init() {
	AddHelp(
		"🛡️ BioMode",
		"biomode_help",
		"🛡️ <b>BioMode</b> monitors user bios and deletes messages if they contain URLs.\n\n"+
			"<b>Usage:</b>\n"+
			"➤ <code>/biolink on</code> - Enable BioMode\n"+
			"➤ <code>/biolink off</code> - Disable BioMode\n\n"+
			"🚫 When enabled, users with links in their bios won't be able to send messages.\n"+
			"👮 Only admins can enable or disable this feature.",
	)
}

func ShouldDeleteMsg(text string) bool {
	pattern := `\b(?:https?://|www\.)\S+|\b[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(?:/\S*)?|\B@\w{5,32}\b|\b\w{5,32}\.t\.me\b`
	re := regexp.MustCompile(pattern)
	return re.MatchString(text)
}

func setBioMode(m *telegram.NewMessage) error {
	if isgroup := IsValidSupergroup(m); !isgroup {
		return nil
	}

	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> bioLink -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> biolink -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")

		return telegram.EndGroup
	}

	args := strings.Fields(m.Text())

	if len(args) < 2 {
		if isEn, err := database.GetBioMode(m.ChatID()); err != nil {
			m.Respond("📚 Usage: <code>/biolink on</code> | <code>/biolink off</code>")
		} else {
			m.Respond("Currently bioLink mode is " + map[bool]string{true: "Enabled", false: "Disabled"}[isEn] + " for your chat")
		}
		// m.Respond("📚 Usage: <code>/biolink on</code> | <code>/biolink off</code>")
		return telegram.EndGroup
	}

	part := args[1]
	var msg string

	if part == "on" || part == "enable" {
		err := database.SetBioMode(m.ChatID())
		if err != nil {
			msg = fmt.Sprintf("⚠️ <b>Oops! Failed to enable BioMode.</b>\n\n🚫 An error occurred while trying to turn it on.\n\n<b>Error:</b> <code>%v</code>\n\n🔁 Please try again later.", err)
			L(m, "Modules -> biolink -> database.SetBioMode(...)", err)

		} else {
			msg = "✅ <b>BioMode enabled successfully!</b>\n\n🔍 I will now monitor bios for any links and automatically delete messages if found.\n\n🛡 Stay safe!"
		}
	} else if part == "off" || part == "disable" {
		err := database.DelBioMode(m.ChatID())
		if err != nil {
			msg = fmt.Sprintf("⚠️ <b>Oops! Failed to disable BioMode.</b>\n\n🚫 An error occurred while trying to turn it off.\n\n<b>Error:</b> <code>%v</code>\n\n🔁 Please try again later.", err)
			L(m, "Modules -> biolink -> database.DelBioMode(...)", err)

		} else {
			msg = "🛑 <b>BioMode disabled successfully!</b>\n\n🔓 I'm no longer monitoring user bios for links in this group.\n\n✅ You're back to normal behavior."
		}
	} else {
		msg = "❗ Invalid option. Use <code>on</code> or <code>off</code>."
	}

	m.Respond(msg)
	return telegram.EndGroup
}

func deleteUserMsgIfBio(m *telegram.NewMessage) error {
	if !IsSupergroup(m) {
		return nil
	}

	mode, err := database.GetBioMode(m.ChatID())
	if err != nil {
		L(m, "Modules -> biolink -> database.GetBioMode(...)", err)
		return err
	}
	if !mode {
		return Continue
	}

	isAdmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID())
	if err != nil {
		L(m, "Modules -> biolink -> utils.IsChatAdmin()", err)
		return err
	}
	if isAdmin {
		return telegram.EndGroup
	}

	if _, ok := m.Message.FromID.(*telegram.PeerUser); !ok {
		return nil
	}

	var bio string
	cacheKey := fmt.Sprintf("userfull_%d", m.SenderID())

	resp, err := m.Client.UsersGetFullUser(&telegram.InputUserObj{
		UserID:     m.Sender.ID,
		AccessHash: m.Sender.AccessHash,
	})

	if err != nil {
		if val, ok := utils.LoadTyped[*telegram.UserFull](bioCache, cacheKey); ok {
			bio = val.About
		} else {
			if wait := telegram.GetFloodWait(err); wait > 0 && wait < 15 {
				time.Sleep(time.Duration(wait) * time.Second)
				resp, err = m.Client.UsersGetFullUser(&telegram.InputUserObj{
					UserID:     m.Sender.ID,
					AccessHash: m.Sender.AccessHash,
				})
				if err != nil {
					return telegram.EndGroup
				}
				bio = resp.FullUser.About
				bioCache.Set(cacheKey, resp.FullUser)
			} else {
				if !strings.Contains(err.Error(), "USER_ID_INVALID") && !strings.Contains(err.Error(), "FLOOD_WAIT_X") {
					L(m, "Modules -> biolink -> client.UsersGetFullUser(...)", err)
				}
				return telegram.EndGroup
			}
		}
	} else {
		bio = resp.FullUser.About
		bioCache.Set(cacheKey, resp.FullUser)
	}

	if bio == "" || !ShouldDeleteMsg(bio) {
		return nil
	}

	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	}

	var mention string
	if m.Sender.Username != "" {
		mention = "@" + m.Sender.Username
	} else {
		mention = fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", m.Sender.ID, m.Sender.FirstName)
	}

	msg := fmt.Sprintf("🚨 %s, your message was deleted because your bio contains a link.", mention)
	return m.E(m.Respond(msg))
}
