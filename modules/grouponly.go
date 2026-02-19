package modules

import (
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func handleNeedPerm(e error, m *telegram.NewMessage) bool {
	if e == nil {
		return false
	} else if strings.Contains(e.Error(), "MESSAGE_DELETE_FORBIDDEN") {
		m.Respond("I need 'Delete Message' Permission to work properly")
		return true
	}
	return false
}

func IsSupergroup(m *telegram.NewMessage) bool {
	return m.ChatType() == telegram.EntityChat && m.Channel != nil
}

func IsBasicGroup(m *telegram.NewMessage) bool {
	return m.ChatType() == telegram.EntityChat && m.Channel == nil
}

func IsAnonymousAdmin(m *telegram.NewMessage) bool {
	if m == nil || m.Message == nil {
		return false
	}
	if m.Message.FromID == nil {
		return true
	}
	if _, isChannel := m.Message.FromID.(*telegram.PeerChannel); isChannel {
		return true
	}
	return false
}

func IsValidSupergroup(m *telegram.NewMessage) bool {
	if !IsSupergroup(m) {
		m.Delete()
		m.Respond("⚠️ This command is only usable in supergroups!")
		return false
	}
	if _, isChannel := m.Message.FromID.(*telegram.PeerChannel); isChannel {
		return false
	}
	if IsAnonymousAdmin(m) {
		m.Delete()
		m.Respond("🚫 You are an anonymous admin. You can't use this command.")
		return false
	}
	return true
}

func ShouldIgnoreGroupAnonymous(m *telegram.NewMessage) bool {
	if !IsAnonymousAdmin(m) {
		return false
	}

	if m.SenderID() == m.ChatID() || m.SenderID() == 0 {
		return true
	}

	fullChat, err := utils.GetFullChannel(m.Client, m.ChatID())
	if err != nil {
		L(m, "Modules -> groupOnly -> GetFullChannel", err)
		return false
	}
	// Check if it's a linked channel message
	if m.SenderID() == fullChat.LinkedChatID {
		return true
	}

	return false
}
