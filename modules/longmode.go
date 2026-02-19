package modules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
	"main/database"
)

func SetLongLimitHandler(m *telegram.NewMessage) error {
	if !IsValidSupergroup(m) {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> longmode -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> longmode -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")

		return telegram.EndGroup
	}

	if m.Args() == "" {
		m.Respond("Usage: /setlonglimit &lt;number&gt;")
		return telegram.EndGroup
	}

	args := strings.Fields(m.Args())
	limit, convErr := strconv.Atoi(args[0])
	if convErr != nil || limit < 200 || limit > 4000 {
		m.Respond("Please provide a valid number between 200 and 4000.")
		return telegram.EndGroup
	}

	err := database.SetEchoLimit(m.ChatID(), limit)
	if err != nil {
		m.Respond(fmt.Sprintf("Error saving limit: %v", err))
		return telegram.EndGroup
	}

	m.Respond(fmt.Sprintf("✅ Long message limit set to %d characters.", limit))
	return telegram.EndGroup
}

func SetLongModeHandler(m *telegram.NewMessage) error {
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> longmode -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> longmode -> IsChatAdmin", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")

		return telegram.EndGroup
	}

	if m.Args() == "" {
		m.Respond("Usage: /setlongmode &lt;off|manual|automatic&gt;")
		return telegram.EndGroup
	}
	mode := strings.ToLower(strings.Fields(m.Args())[0])
	if mode != "off" && mode != "manual" && mode != "automatic" && mode != "auto" {
		m.Respond("Invalid mode. Use one of: <code>off</code>, <code>manual</code>, <code>automatic</code>.")
		return telegram.EndGroup
	}

	if mode == "auto" {
		mode = "automatic"
	}

	err := database.SetEchoMode(m.ChatID(), mode)
	if err != nil {
		return L(m, "Modules -> longmode -> SetEchoMode", err)
	}

	m.Respond(fmt.Sprintf("✅ Long message mode set to '%s'.", mode))
	return telegram.EndGroup
}
