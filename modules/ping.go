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
	_, _ = m.Delete()
	latency := time.Since(start).Milliseconds()
	uptime := time.Since(config.StartTime)
	uptimeStr := utils.FormatUptime(uptime)

	text := fmt.Sprintf(
		"<b>🏓 Pong!</b>\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"🚀 <b>Latency:</b> <code>%dms</code>\n"+
			"🆙 <b>Uptime:</b> <code>%s</code>",
		latency,
		uptimeStr,
	)

	opts := &telegram.SendOptions{
		ParseMode: "HTML",
	}

	if m.IsPrivate() {
		btn := telegram.NewKeyboard().AddRow(
			telegram.Button.Data("🏠 Back to Menu", "start_callback"),
		).Build()
		opts.ReplyMarkup = btn
	}

	_, err := m.Reply(text, opts)
	return err
}
