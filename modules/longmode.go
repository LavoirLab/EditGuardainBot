package modules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "longmode",
		ModName: "📏 LongMode",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:setlonglimit",
				Handler: RequirePermission("change_info", SetLongLimitHandler),
			},
			{
				Pattern: "command:setlongmode",
				Handler: RequirePermission("change_info", SetLongModeHandler),
			},
		},
	})
}

func SetLongLimitHandler(m *telegram.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err := m.Reply("⚠️ <b>Usage:</b> <code>/setlonglimit &lt;number&gt;</code>\n\n<b>Example:</b> <code>/setlonglimit 1000</code>\n\n<i>Valid range: 200 to 4000 characters.</i>")
		return err
	}

	args := strings.Fields(m.Args())
	limit, convErr := strconv.Atoi(args[0])
	if convErr != nil || limit < 200 || limit > 4000 {
		_, err := m.Reply("❌ <b>Invalid limit.</b>\n\nPlease provide a number between <code>200</code> and <code>4000</code>.")
		return err
	}

	err := database.SetEchoLimit(m.ChatID(), limit)
	if err != nil {
		_, err = m.Reply("⚠️ <b>Failed to save limit.</b>\n\nSomething went wrong while updating the character limit. Please try again.")
		return err
	}

	_, err = m.Reply(
		fmt.Sprintf("✅ <b>Long message limit set to %d characters!</b>\n\nMessages exceeding this length will be handled based on the current mode.", limit),
	)
	return err
}

func SetLongModeHandler(m *telegram.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err := m.Reply("⚠️ <b>Usage:</b> <code>/setlongmode &lt;mode&gt;</code>\n\n<b>Available Modes:</b>\n• <code>off</code> — No action on long messages\n• <code>manual</code> — Warn users to use /echo\n• <code>automatic</code> — Automatically convert to Telegraph links")
		return err
	}
	mode := strings.ToLower(strings.Fields(m.Args())[0])
	if mode != "off" && mode != "manual" && mode != "automatic" &&
		mode != "auto" {
		_, err := m.Reply(
			"❌ <b>Invalid mode.</b>\n\nPlease use one of: <code>off</code>, <code>manual</code>, or <code>automatic</code>.",
		)
		return err
	}

	if mode == "auto" {
		mode = "automatic"
	}

	err := database.SetEchoMode(m.ChatID(), mode)
	if err != nil {
		_, err = m.Reply("⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the long message mode. Please try again.")
		return err
	}

	_, err = m.Reply(fmt.Sprintf("✅ <b>Long message mode set to '%s'!</b>\n\nThis will now apply to all long messages in the chat.", mode))
	return err
}
