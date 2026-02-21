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
	"main/utils"
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

	if config.NsfwApiUrl != "" {
		if err := utils.CheckNsfwApi(config.NsfwApiUrl); err != nil {
			log.Printf("⚠️ NSFW API check failed: %v. NSFW features will be disabled.", err)
			config.NsfwApiEnabled = false
		} else {
			log.Println("✅ NSFW API is online and working.")
			config.NsfwApiEnabled = true
		}
	} else {
		log.Println("ℹ️ NSFW API URL not set. NSFW features will be disabled.")
		config.NsfwApiEnabled = false
	}

	modules.LoadAllHandlers(client)

	client.SendMessage(config.LoggerId, "Started...")
	log.Println("✅ Bot Started")
	client.Idle()
}
