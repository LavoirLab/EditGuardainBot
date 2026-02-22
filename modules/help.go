package modules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "help",
		ModName: "📚 Help",
		Help:    "",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:help",
				Handler: handleHelp,
			},
			{
				Pattern: "callback:^help$",
				Handler: handleHelpCallback,
			},
			{
				Pattern: "callback:^help_module:",
				Handler: handleHelpModuleCallback,
			},
		},
	})
}

const helpText = `📚 <b>Bot Command Help</b>

Here you'll find details for all available plugins and features.

👇 Tap the buttons below to view help for each module:`

func handleHelp(m *telegram.NewMessage) error {
	if !m.IsPrivate() {
		keyboard := telegram.NewKeyboard().AddRow(
			telegram.Button.URL("📖 Open Help in PM", fmt.Sprintf("https://t.me/%s?start=help", m.Client.Me().Username)),
		).
			Build()

		m.Reply(
			"👋 <b>Need help?</b>\n\nTo see all available commands and how to use them, please contact me in private message.",
			&telegram.SendOptions{
				ReplyMarkup: keyboard,
			},
		)

		return telegram.ErrEndGroup
	}

	m.Reply(helpText, &telegram.SendOptions{
		ReplyMarkup: getFullHelp(),
	})
	return telegram.ErrEndGroup
}

func handleHelpCallback(c *telegram.CallbackQuery) error {
	c.Answer("")
	c.Edit(helpText, &telegram.SendOptions{ReplyMarkup: getFullHelp()})
	return telegram.ErrEndGroup
}

func getFullHelp() telegram.ReplyMarkup {
	keyboard := telegram.NewKeyboard()

	modules := GetModulesWithHelp()

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})

	var buttons []telegram.KeyboardButton

	for _, mod := range modules {
		if mod.Name == "help" || mod.Name == "start" {
			continue
		}
		button := telegram.Button.Data(mod.ModName, "help_module:"+mod.Name)
		buttons = append(buttons, button)
	}

	keyboard.NewColumn(3, buttons...)
	keyboard.AddRow(telegram.Button.Data("⬅️ Back", "start_callback"))

	return keyboard.Build()
}

func handleHelpModuleCallback(c *telegram.CallbackQuery) error {
	data := c.DataString()
	c.Answer("")
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	moduleName := parts[1]

	module, exists := GetModule(moduleName)
	modHelpText := "❌ No help found for this module."
	if exists && module.Help != "" {
		modHelpText = "📖 <b>Help for " + module.ModName + "</b>\n\n" + module.Help
	}

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data("⬅️ Back", "help"),
	).Build()

	_, err := c.Edit(modHelpText, &telegram.SendOptions{
		ReplyMarkup: keyboard,
	})
	return err
}
