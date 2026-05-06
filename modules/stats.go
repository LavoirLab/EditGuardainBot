package modules

import (
	"fmt"
	"runtime"
	"slices"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"

	"main/config"
	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "stats",
		ModName: "📊 Stats",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:stats",
				Handler: stats,
			},
		},
	})
}

func stats(msg *telegram.NewMessage) error {
	if _, err := msg.Delete(); err != nil {
		return err
	}

	if !slices.Contains(config.OwnerId, msg.SenderID()) {
		text := "❌ <b>Access Denied</b>\n\n" +
			"<i>This command is reserved for the bot owner only.</i>"

		opts := &telegram.SendOptions{
			ParseMode: "HTML",
		}

		if msg.IsPrivate() {
			btn := telegram.NewKeyboard().AddRow(
				telegram.Button.Data("🏠 Back to Menu", "start_callback"),
			).Build()
			opts.ReplyMarkup = btn
		}

		_, err := msg.Reply(text, opts)
		return err
	}

	var (
		chatCount int
		userCount int
		ramInfo   string
		diskInfo  string
		cpuInfo   string
		uptimeStr = utils.FormatUptime(time.Since(config.StartTime))
	)

	// Optimization: Use dedicated O(1) count functions instead of fetching
	// and allocating memory for all IDs just to count them.
	if count, err := database.GetServedChatsCount(); err == nil {
		chatCount = count
	}
	if count, err := database.GetServedUsersCount(); err == nil {
		userCount = count
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		ramInfo = fmt.Sprintf(
			"<code>%.1f%%</code> of <code>%.1f GB</code>",
			vm.UsedPercent,
			float64(vm.Total)/1e9,
		)
	} else {
		ramInfo = "<code>N/A</code>"
	}

	if d, err := disk.Usage("/"); err == nil {
		diskInfo = fmt.Sprintf(
			"<code>%.1f%%</code> of <code>%.1f GB</code>",
			d.UsedPercent,
			float64(d.Total)/1e9,
		)
	} else {
		diskInfo = "<code>N/A</code>"
	}

	if percent, err := cpu.Percent(0, false); err == nil && len(percent) > 0 {
		cpuInfo = fmt.Sprintf("<code>%.2f%%</code>", percent[0])
	} else {
		cpuInfo = "<code>N/A</code>"
	}

	text := fmt.Sprintf(
		`📊 <b>Bot Statistics</b>
━━━━━━━━━━━━━━━━━━━━
💬 <b>Total Chats:</b> <code>%d</code>
👤 <b>Total Users:</b> <code>%d</code>
⏱️ <b>Uptime:</b> <code>%s</code>

🖥️ <b>Performance</b>
<blockquote>🧠 <b>RAM:</b> %s
💾 <b>Disk:</b> %s
⚡ <b>CPU:</b> %s</blockquote>

🛠️ <b>Technical Details</b>
<blockquote expandable>• <b>OS:</b> <code>%s</code> (<code>%s</code>)
• <b>CPUs:</b> <code>%d</code>
• <b>Goroutines:</b> <code>%d</code></blockquote>`,
		chatCount,
		userCount,
		uptimeStr,
		ramInfo,
		diskInfo,
		cpuInfo,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		runtime.NumGoroutine(),
	)

	opts := &telegram.SendOptions{
		ParseMode: "HTML",
	}

	if msg.IsPrivate() {
		btn := telegram.NewKeyboard().AddRow(
			telegram.Button.Data("🏠 Back to Menu", "start_callback"),
		).Build()
		opts.ReplyMarkup = btn
	}

	_, err := msg.Reply(text, opts)
	return err
}
