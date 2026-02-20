package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/database"
	"main/modules"
)

func main() {
	defer database.Disconnect()

	client, err := telegram.NewClient(telegram.ClientConfig{
		AppID:     config.ApiId,
		AppHash:   config.ApiHash,
		LogLevel:  telegram.LogDebug,
		ParseMode: "HTML",
	})
	if err != nil {
		log.Panic(err)
	}

	if err := client.LoginBot(config.Token); err != nil {
		if strings.Contains(err.Error(), "ACCESS_TOKEN_EXPIRED") {
			fmt.Println("❌ Bot token has been revoked or expired.")
			os.Exit(1)
		}
		log.Panic(err)
	}

	modules.BotInfo, err = client.GetMe()
	if err != nil {
		client.SendMessage(config.LoggerId, "Failed to getMe: "+err.Error())
	}

	modules.LoadAllHandlers(client)

	client.SendMessage(config.LoggerId, "Started...")
	log.Println("✅ Bot Started")
	client.Idle()
}
