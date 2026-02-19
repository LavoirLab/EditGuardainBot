package utils

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils/cache"
)

var adminsCache = cache.NewCache[string, any](30 * time.Minute)

func SetAdminsCache(chatID int64, admins []*telegram.Participant) {
	adminsCache.Set(fmt.Sprintf("admins:%d", chatID), admins)
}

func FetchAdmins(b *telegram.Client, ChatId int64) ([]*telegram.Participant, error) {
	cacheKey := fmt.Sprintf("admins:%d", ChatId)

	if admins, ok := LoadTyped[[]*telegram.Participant](adminsCache, cacheKey); ok {
		return admins, nil
	}
	admins, _, err := b.GetChatMembers(ChatId, &telegram.ParticipantOptions{
		Filter: &telegram.ChannelParticipantsAdmins{},
		Limit:  -1,
	})
	if err != nil {
		return nil, err
	}
	adminsCache.Set(cacheKey, admins)
	return admins, nil
}

func GetAdmins(b *telegram.Client, ChatId int64) ([]int64, error) {
	admins, err := FetchAdmins(b, ChatId)
	if err != nil {
		return nil, err
	}

	var ids []int64
	for _, p := range admins {
		if p.User.Bot || p.User.Deleted {
			continue
		}
		ids = append(ids, p.User.ID)
	}

	return ids, nil
}

func IsChatAdmin(c *telegram.Client, chatid, userid int64) (bool, error) {
	ids, err := GetAdmins(c, chatid)
	if err != nil {
		return false, err
	}
	return slices.Contains(ids, userid), nil
}

func GetOwner(b *telegram.Client, ChatId int64) (int64, error) {
	admins, err := FetchAdmins(b, ChatId)
	if err != nil {
		return 0, err
	}

	for _, p := range admins {
		if _, ok := p.Participant.(*telegram.ChannelParticipantCreator); ok {
			return p.User.ID, nil
		}
	}

	return 0, fmt.Errorf("no creator found")
}

func WarnIfLackOfPms(b *telegram.Client, m *telegram.NewMessage, chatid int64) bool {
	admins, _, err := b.GetChatMembers(chatid, &telegram.ParticipantOptions{
		Filter: &telegram.ChannelParticipantsAdmins{},
		Limit:  -1,
	})
	if err != nil {
		log.Printf("Failed to get admins list for chat %d: %v", chatid, err)
		if strings.Contains(err.Error(), "CHAT_ADMIN_REQUIRED") {
			m.Respond("⚠️ Please promote me to <b>Admin</b> with <b>\"Delete messages\" permission</b> so I can protect this group properly.")
		} else {
			m.Respond("❌ Failed to fetch admin list. Ensure I have permission to see group admins.")
		}
		return false
	}

	adminsCache.Set(fmt.Sprintf("admins:%d", chatid), admins)

	botID := b.Me().ID
	for _, p := range admins {
		if p.User.ID == botID {
			if p.Status != "admin" && p.Status != "creator" {
				m.Respond("⚠️ Please promote me to <b>Admin</b> with <b>\"Delete messages\" permission</b> so I can protect this group properly.")
				return false
			}
			if p.Rights == nil || !p.Rights.DeleteMessages {
				m.Respond("🧹 <b>I need the \"Delete messages\" permission</b> to remove spam or unwanted content.")
				return false
			}
			return true
		}
	}

	log.Printf("Bot not found in admin list for chat %d", chatid)
	m.Respond("⚠️ I am not an admin in this group. Please promote me with necessary permissions.")
	return false
}
