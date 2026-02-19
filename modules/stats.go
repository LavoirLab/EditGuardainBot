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
		ModName: "Stats",
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
		_, err := msg.Reply("You are not authorised to use this command.")
		return err
	}

	var text string

	if chats, err := database.GetServedChats(); err == nil {
		text += fmt.Sprintf("💬 <b>Total Chats:</b> %d\n", len(chats))
	}
	if users, err := database.GetServedUsers(); err == nil {
		text += fmt.Sprintf("👤 <b>Total Users:</b> %d\n", len(users))
	}

	uptime := time.Since(config.StartTime)
	text += fmt.Sprintf("⏱️ <b>Bot Uptime:</b> %s\n", utils.FormatUptime(uptime))

	if vm, err := mem.VirtualMemory(); err == nil {
		text += fmt.Sprintf("🧠 <b>RAM:</b> %.1f%% of %.1f GB\n", vm.UsedPercent, float64(vm.Total)/1e9)
	}

	if d, err := disk.Usage("/"); err == nil {
		text += fmt.Sprintf("💾 <b>Disk:</b> %.1f%% of %.1f GB\n", d.UsedPercent, float64(d.Total)/1e9)
	}

	text += fmt.Sprintf("🔧 <b>System:</b>\n")
	text += fmt.Sprintf("• OS: <code>%s</code>, Arch: <code>%s</code>\n", runtime.GOOS, runtime.GOARCH)
	text += fmt.Sprintf("• CPUs: <code>%d</code>, Goroutines: <code>%d</code>\n", runtime.NumCPU(), runtime.NumGoroutine())

	if percent, err := cpu.Percent(0, false); err == nil && len(percent) > 0 {
		text += fmt.Sprintf("• CPU: <code>%.2f%%</code>\n", percent[0])
	}

	_, err := msg.Reply(text)
	return err
}
