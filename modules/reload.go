package modules

import (
	"fmt"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func ReloadHandler(m *telegram.NewMessage) error {
	m.Delete()
	if !IsValidSupergroup(m) {
		return telegram.EndGroup
	}

	x, _ := m.Respond("Refreshing Cache of chat admins...")

	admins, _, err := m.Client.GetChatMembers(m.ChatID(), &telegram.ParticipantOptions{
		Filter: &telegram.ChannelParticipantsAdmins{},
		Limit:  -1,
	})
	if err != nil {
		if strings.Contains(err.Error(), "CHAT_ADMIN_REQUIRED") {
			x.Edit("⚠️ I am not an admin in this group. Please promote me with necessary permissions.")
			return telegram.EndGroup
		}
		x.Edit(fmt.Sprintf("⚠️ Cache refresh failed — %v", err))

		return L(m, "Modules -> Reload -> GetChatMembers", err)

	}
	utils.SetAdminsCache(m.ChatID(), admins)

	var text string
	if isb, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return err
	} else if isb {
		text = "✅ Successfully refreshed the cache of chat admins!"
	} else {
		text = "⚠️ Tried refreshing the admin cache... but it still seems you're not an admin!"
	}

	return m.E(x.Edit(text))
}
