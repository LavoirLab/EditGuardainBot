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
		ModName: "Long Mode",
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
		_, err := m.Reply("Usage: /setlonglimit &lt;number&gt;")
		return err
	}

	args := strings.Fields(m.Args())
	limit, convErr := strconv.Atoi(args[0])
	if convErr != nil || limit < 200 || limit > 4000 {
		_, err := m.Reply("Please provide a valid number between 200 and 4000.")
		return err
	}

	err := database.SetEchoLimit(m.ChatID(), limit)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf("Error saving limit: %v", err))
		return err
	}

	_, err = m.Reply(fmt.Sprintf("✅ Long message limit set to %d characters.", limit))
	return err
}

func SetLongModeHandler(m *telegram.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}

	if m.Args() == "" {
		_, err := m.Reply("Usage: /setlongmode &lt;off|manual|automatic&gt;")
		return err
	}
	mode := strings.ToLower(strings.Fields(m.Args())[0])
	if mode != "off" && mode != "manual" && mode != "automatic" && mode != "auto" {
		_, err := m.Reply("Invalid mode. Use one of: <code>off</code>, <code>manual</code>, <code>automatic</code>.")
		return err
	}

	if mode == "auto" {
		mode = "automatic"
	}

	err := database.SetEchoMode(m.ChatID(), mode)
	if err != nil {
		return err
	}

	_, err = m.Reply(fmt.Sprintf("✅ Long message mode set to '%s'.", mode))
	return err
}
