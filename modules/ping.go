package modules

import (
	"fmt"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "ping",
		ModName: "🏓 Ping",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:ping",
				Handler: pingHandler,
			},
		},
	})
}

func pingHandler(m *telegram.NewMessage) error {
	start := time.Now()
	reply, err := m.Reply("🏓 Pinging...")
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()
	uptime := time.Since(config.StartTime)
	uptimeStr := utils.FormatUptime(uptime)

	text := fmt.Sprintf(
		"🏓 Pong!\nLatency: %dms\n🤖 I've been running for %s without rest!",
		latency,
		uptimeStr,
	)

	_, err = reply.Edit(text)
	return err
}
