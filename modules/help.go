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

const helpText = `<emoji id='5814427657609153890'>📚</emoji> <b>ʙᴏᴛ ᴄᴏᴍᴍᴀɴᴅ ʜᴇʟᴘ</b>

ʜᴇʀᴇ ʏᴏᴜ'ʟʟ ꜰɪɴᴅ ᴅᴇᴛᴀɪʟꜱ ꜰᴏʀ ᴀʟʟ ᴀᴠᴀɪʟᴀʙʟᴇ ᴘʟᴜɢɪɴꜱ ᴀɴᴅ ꜰᴇᴀᴛᴜʀᴇꜱ.

<b><i><emoji id='5411322793373477644'>👇</emoji> ᴛᴀᴘ ᴛʜᴇ ʙᴜᴛᴛᴏɴꜱ ʙᴇʟᴏᴡ ᴛᴏ ᴠɪᴇᴡ ʜᴇʟᴘ ꜰᴏʀ ᴇᴀᴄʜ ᴍᴏᴅᴜʟᴇ:</i></b>`

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
		modHelpText = module.Help
	}

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data("⬅️ Back", "help"),
	).Build()

	_, err := c.Edit(modHelpText, &telegram.SendOptions{
		ReplyMarkup: keyboard,
	})
	return err
}
