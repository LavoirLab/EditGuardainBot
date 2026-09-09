package utils

import (
	"fmt"
	"log"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

type AdminInfo struct {
	UserID    int64
	Rights    *telegram.ChatAdminRights
	IsCreator bool
	Rank      string
}

type AdminListCache struct {
	Admins map[int64]*AdminInfo
}

type adminCacheKey struct {
	chatID int64
	userID int64
}

var (
	adminCache           = NewCache[int64, *AdminListCache](2 * time.Hour)
	individualAdminCache = NewCache[adminCacheKey, *AdminInfo](2 * time.Hour)
)

func GetChatAdmins(
	client *telegram.Client,
	chatID int64,
) (map[int64]*AdminInfo, error) {
	if cached, ok := adminCache.Get(chatID); ok {
		return cached.Admins, nil
	}

	return fetchAndCacheAdmins(client, chatID)
}

func fetchAndCacheAdmins(
	client *telegram.Client,
	chatID int64,
) (map[int64]*AdminInfo, error) {
	participants, _, err := client.GetChatMembers(
		chatID,
		&telegram.ParticipantOptions{
			Filter: &telegram.ChannelParticipantsAdmins{},
			Limit:  -1,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch admins: %w", err)
	}

	admins := make(map[int64]*AdminInfo)

	for _, participant := range participants {
		if participant.User == nil {
			continue
		}

		adminInfo := &AdminInfo{
			UserID:    participant.User.ID,
			Rights:    participant.Rights,
			IsCreator: participant.Status == telegram.Creator,
			Rank:      participant.Rank,
		}

		admins[participant.User.ID] = adminInfo
		individualAdminCache.Set(
			adminCacheKey{chatID, participant.User.ID},
			adminInfo,
		)
	}

	adminCache.Set(chatID, &AdminListCache{Admins: admins})

	return admins, nil
}

// GetAdmin returns (*AdminInfo, nil) if user is admin,
// (nil, nil) if user is NOT admin,
// or (nil, err) on failure.
func GetAdmin(
	client *telegram.Client,
	chatID, userID int64,
) (*AdminInfo, error) {
	cacheKey := adminCacheKey{chatID, userID}
	if cached, ok := individualAdminCache.Get(cacheKey); ok {
		return cached, nil
	}

	admins, err := GetChatAdmins(client, chatID)
	if err != nil {
		return nil, err
	}

	if admin, exists := admins[userID]; exists {
		return admin, nil
	}

	return nil, nil
}

func IsAdmin(client *telegram.Client, chatID, userID int64) (bool, error) {
	cacheKey := adminCacheKey{chatID, userID}
	if _, ok := individualAdminCache.Get(cacheKey); ok {
		return true, nil
	}

	admins, err := GetChatAdmins(client, chatID)
	if err != nil {
		return false, err
	}

	_, exists := admins[userID]
	return exists, nil
}

func CheckUserPermission(
	client *telegram.Client,
	chatID, userID int64,
	permission string,
) (bool, bool, error) {
	admin, err := GetAdmin(client, chatID, userID)
	if err != nil {
		return false, false, err
	}

	if admin == nil {
		return false, false, nil
	}
	if admin.Rights == nil {
		return false, true, nil
	}

	hasPermission := checkPermission(admin.Rights, permission)
	return hasPermission, true, nil
}

func checkPermission(rights *telegram.ChatAdminRights, permission string) bool {
	if rights == nil {
		return false
	}

	switch permission {
	case "change_info":
		return rights.ChangeInfo
	case "post_messages":
		return rights.PostMessages
	case "edit_messages":
		return rights.EditMessages
	case "delete_messages":
		return rights.DeleteMessages
	case "ban_users":
		return rights.BanUsers
	case "invite_users":
		return rights.InviteUsers
	case "pin_messages":
		return rights.PinMessages
	case "add_admins":
		return rights.AddAdmins
	case "manage_call":
		return rights.ManageCall
	case "anonymous":
		return rights.Anonymous
	case "manage_topics":
		return rights.ManageTopics
	case "post_stories":
		return rights.PostStories
	case "edit_stories":
		return rights.EditStories
	case "delete_stories":
		return rights.DeleteStories
	default:
		return false
	}
}

func UpdateAdminFromParticipant(
	client *telegram.Client,
	chatID, userID int64,
	participant telegram.ChannelParticipant,
) {
	if participant == nil {
		return
	}

	adminInfo := &AdminInfo{
		UserID: userID,
	}

	switch p := participant.(type) {
	case *telegram.ChannelParticipantCreator:
		adminInfo.IsCreator = true
		adminInfo.Rights = p.AdminRights
		adminInfo.Rank = p.Rank

	case *telegram.ChannelParticipantAdmin:
		adminInfo.IsCreator = false
		adminInfo.Rights = p.AdminRights
		adminInfo.Rank = p.Rank
	default:
		RemoveAdmin(chatID, userID)
		return
	}

	cacheKey := adminCacheKey{chatID, userID}
	individualAdminCache.Set(cacheKey, adminInfo)

	if cached, ok := adminCache.Get(chatID); ok {
		cached.Admins[userID] = adminInfo
		adminCache.Set(chatID, cached)
	}
}

func RemoveAdmin(chatID, userID int64) {
	cacheKey := adminCacheKey{chatID, userID}
	individualAdminCache.Delete(cacheKey)

	if cached, ok := adminCache.Get(chatID); ok {
		delete(cached.Admins, userID)
		adminCache.Set(chatID, cached)
	}
}

func ClearChatAdminCache(chatID int64) {
	adminCache.Delete(chatID)

	for _, key := range individualAdminCache.Keys() {
		if key.chatID == chatID {
			individualAdminCache.Delete(key)
		}
	}

	log.Printf("Cleared admin cache (chat + individual) for chat %d", chatID)
}

func ClearAllAdminCache() {
	adminCache.Clear()
	individualAdminCache.Clear()
	log.Println("Cleared entire admin cache")
}

func RefreshAdminCache(client *telegram.Client, chatID int64) error {
	_, err := fetchAndCacheAdmins(client, chatID)
	return err
}

func PrettifyPermission(permission string) string {
	permissionMap := map[string]string{
		"change_info":     "change chat info",
		"post_messages":   "post messages",
		"edit_messages":   "edit messages",
		"delete_messages": "delete messages",
		"ban_users":       "ban users",
		"invite_users":    "invite users",
		"pin_messages":    "pin messages",
		"add_admins":      "add admins",
		"manage_call":     "manage calls",
		"anonymous":       "remain anonymous",
		"manage_topics":   "manage topics",
		"post_stories":    "post stories",
		"edit_stories":    "edit stories",
		"delete_stories":  "delete stories",
	}

	if pretty, exists := permissionMap[permission]; exists {
		return pretty
	}
	return permission
}

func GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_duration_minutes": 5.0,
		"note":                   "Using generic cache with automatic expiry",
	}
}
