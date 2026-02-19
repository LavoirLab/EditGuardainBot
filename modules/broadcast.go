package modules

import (
	"fmt"
	"log"
	"slices"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "broadcast",
		ModName: "Broadcast",
		Help:    "",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:gcast",
				Handler: BroadcastFunc,
			},
		},
	})
}

func BroadcastFunc(m *telegram.NewMessage) error {
	if !slices.Contains(config.OwnerId, m.SenderID()) {
		return telegram.EndGroup
	}

	userChan, chatChan, err := m.Client.Broadcast()
	if err != nil {
		log.Println(err)
		_, err := m.Reply(err.Error())
		return err
	}

	userCount := 0
	for range userChan {
		userCount++
	}

	chatCount := 0
	for range chatChan {
		chatCount++
	}
	m.Delete()
	_, err = m.Reply(fmt.Sprintf("Total Chats: %d\nTotal Users: %d", chatCount, userCount))
	if err != nil {
		return err
	}
	_, err = m.Reply("Soon implemented....")
	return err
}
