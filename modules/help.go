package modules

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/amarnathcjd/gogram/telegram"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "help",
		ModName: "Help",
		Help:    "Contact me in PM for help!",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:help",
				Handler: handleHelp,
			},
			{
				Pattern: "callback:help",
				Handler: handleHelpCallback,
			},
			{
				Pattern: "callback:help_module",
				Handler: handleHelpModuleCallback,
			},
		},
	})
}

func handleHelp(m *telegram.NewMessage) error {
	m.Delete()
	if !m.IsPrivate() {
		keyboard := telegram.NewKeyboard().AddRow(
			telegram.Button.URL("🗒 Command", fmt.Sprintf("https://t.me/%s?start=help", m.Client.Me().Username)),
		).Build()
		_, err := m.Reply("Contact me in PM for help!", &telegram.SendOptions{
			ReplyMarkup: keyboard,
		})
		return err
	}

	return sendFullHelp(m.Client, m.ChatID(), false)
}

func handleHelpCallback(c *telegram.CallbackQuery) error {
	c.Answer("")
	return sendFullHelp(c.Client, c.ChatID, true)
}

func sendFullHelp(client *telegram.Client, chatID int64, isEdit bool) error {
	keyboard := telegram.NewKeyboard()

	modules := GetModulesWithHelp()

	sort.Slice(modules, func(i, j int) bool {
		return stripPrefixEmoji(modules[i].ModName) < stripPrefixEmoji(modules[j].ModName)
	})

	var buttons []telegram.KeyboardButton

	for _, mod := range modules {
		if mod.Name == "help" || mod.Name == "start" {
			continue
		}
		button := telegram.Button.Data(mod.ModName, "help_module:"+mod.Name)
		buttons = append(buttons, button)
	}

	keyboard.NewColumn(2, buttons...)
	keyboard.AddRow(telegram.Button.Data("🗑️ Close", "close"))

	helpText := `📚 <b>Bot Command Help</b>

Here you'll find details for all available plugins and features.

👇 Tap the buttons below to view help for each module:`

	var err error
	if isEdit {
		_, err = client.EditMessage(chatID, 0, helpText, &telegram.SendOptions{
			ReplyMarkup: keyboard.Build(),
		})
	} else {
		_, err = client.SendMessage(chatID, helpText, &telegram.SendOptions{
			ReplyMarkup: keyboard.Build(),
		})
	}
	return err
}

func handleHelpModuleCallback(c *telegram.CallbackQuery) error {
	data := c.DataString()
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	moduleName := parts[1]

	module, exists := GetModule(moduleName)
	helpText := "❌ No help found for this module."
	if exists && module.Help != "" {
		helpText = module.Help
	}

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data("⬅️ Back", "help"),
	).Build()

	_, err := c.Edit(helpText, &telegram.SendOptions{
		ReplyMarkup: keyboard,
	})
	return err
}

func stripPrefixEmoji(s string) string {
	for i, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return strings.ToLower(s[i:])
		}
	}
	return s
}
