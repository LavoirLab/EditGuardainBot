package modules

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
)

const (
	HelpColumns = 3
	HelpRows    = 4
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
				Pattern: "callback:^help_page:(\\d+)$",
				Handler: handleHelpPageCallback,
			},
			{
				Pattern: "callback:^help_module:",
				Handler: handleHelpModuleCallback,
			},
		},
	})
}

const helpText = `<emoji id='5814427657609153890'>📚</emoji> <b>ʙᴏᴛ ᴄᴏᴍᴍᴀɴᴅ ʜᴇʟᴘ</b>
━━━━━━━━━━━━━━━━━━━━
ʜᴇʀᴇ ʏᴏᴜ'ʟʟ ꜰɪɴᴅ ᴅᴇᴛᴀɪʟꜱ ꜰᴏʀ ᴀʟʟ ᴀᴠᴀɪʟᴀʙʟᴇ ᴘʟᴜɢɪɴꜱ ᴀɴᴅ ꜰᴇᴀᴛᴜʀᴇꜱ.

<b><i><emoji id='5411322793373477644'>👇</emoji> ᴛᴀᴘ ᴛʜᴇ ʙᴜᴛᴛᴏɴꜱ ʙᴇʟᴏᴡ ᴛᴏ ᴠɪᴇᴡ ʜᴇʟᴘ ꜰᴏʀ ᴇᴀᴄʜ ᴍᴏᴅᴜʟᴇ:</i></b>`

func handleHelp(m *telegram.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "nsfw" && !config.NsfwApiEnabled {
		m.Reply(
			"That module is disabled.",
			&telegram.SendOptions{ParseMode: "HTML"},
		)
		return telegram.ErrEndGroup
	}

	if !m.IsPrivate() {
		keyboard := telegram.NewKeyboard().AddRow(
			telegram.Button.URL("📖 Open Help in PM", fmt.Sprintf("https://t.me/%s?start=help", m.Client.Me().Username)),
		).
			Build()

		m.Reply(
			"👋 <b>ɴᴇᴇᴅ ʜᴇʟᴘ?</b>\n\n"+
				"<i>ᴛᴏ ꜱᴇᴇ ᴀʟʟ ᴀᴠᴀɪʟᴀʙʟᴇ ᴄᴏᴍᴍᴀɴᴅꜱ ᴀɴᴅ ʜᴏᴡ ᴛᴏ ᴜꜱᴇ ᴛʜᴇᴍ, ᴘʟᴇᴀꜱᴇ ᴄᴏɴᴛᴀᴄᴛ ᴍᴇ ɪɴ ᴘʀɪᴠᴀᴛᴇ ᴍᴇꜱꜱᴀɢᴇ.</i>\n\n"+
				"💡 <b>ᴛɪᴘ:</b> ᴅɪʀᴇᴄᴛ ᴍᴇꜱꜱᴀɢᴇꜱ ᴀʀᴇ ᴜꜱᴇᴅ ᴛᴏ ᴋᴇᴇᴘ ɢʀᴏᴜᴘ ᴄʜᴀᴛꜱ ᴄʟᴇᴀɴ ᴀɴᴅ ᴘʀᴏᴠɪᴅᴇ ᴀ ʙᴇᴛᴛᴇʀ ʜᴇʟᴘ ᴇxᴘᴇʀɪᴇɴᴄᴇ.",
			&telegram.SendOptions{
				ReplyMarkup: keyboard,
				ParseMode:   "HTML",
			},
		)

		return telegram.ErrEndGroup
	}

	m.Reply(helpText, &telegram.SendOptions{
		ReplyMarkup: getHelpKeyboard(0),
		ParseMode:   "HTML",
	})
	return telegram.ErrEndGroup
}

func handleHelpCallback(c *telegram.CallbackQuery) error {
	c.Answer("")
	c.Edit(helpText, &telegram.SendOptions{
		ReplyMarkup: getHelpKeyboard(0),
		ParseMode:   "HTML",
	})
	return telegram.ErrEndGroup
}

func handleHelpPageCallback(c *telegram.CallbackQuery) error {
	c.Answer("")
	data := c.DataString()
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}
	page, _ := strconv.Atoi(parts[1])
	c.Edit(helpText, &telegram.SendOptions{
		ReplyMarkup: getHelpKeyboard(page),
		ParseMode:   "HTML",
	})
	return telegram.ErrEndGroup
}

func getHelpKeyboard(page int) telegram.ReplyMarkup {
	keyboard := telegram.NewKeyboard()

	allModules := GetModulesWithHelp()
	var modules []*ModuleInfo

	for _, mod := range allModules {

		if mod.Name == "help" || mod.Name == "start" {
			continue
		}

		if mod.Name == "nsfw" && !config.NsfwApiEnabled {
			continue
		}

		modules = append(modules, mod)
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})

	buttonsPerPage := HelpColumns * HelpRows
	totalModules := len(modules)

	start := page * buttonsPerPage
	if start >= totalModules && totalModules > 0 {
		page = 0
		start = 0
	}
	end := start + buttonsPerPage
	if end > totalModules {
		end = totalModules
	}

	var buttons []telegram.KeyboardButton
	for i := start; i < end; i++ {
		mod := modules[i]
		button := telegram.Button.Data(
			mod.ModName,
			fmt.Sprintf("help_module:%s:%d", mod.Name, page),
		)
		buttons = append(buttons, button)
	}

	keyboard.NewColumn(HelpColumns, buttons...)

	var navButtons []telegram.KeyboardButton
	if page > 0 {
		navButtons = append(
			navButtons,
			telegram.Button.Data(
				"⬅️ Previous",
				fmt.Sprintf("help_page:%d", page-1),
			),
		)
	}

	navButtons = append(
		navButtons,
		telegram.Button.Data("🏠 Back", "start_callback"),
	)

	if end < totalModules {
		navButtons = append(
			navButtons,
			telegram.Button.Data(
				"➡️ Next",
				fmt.Sprintf("help_page:%d", page+1),
			),
		)
	}

	keyboard.AddRow(navButtons...)

	return keyboard.Build()
}

func handleHelpModuleCallback(c *telegram.CallbackQuery) error {
	data := c.DataString()
	c.Answer("")
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}

	moduleName := parts[1]

	if moduleName == "nsfw" && !config.NsfwApiEnabled {
		c.Answer("This module is not available.")
		return telegram.ErrEndGroup
	}

	page := 0
	if len(parts) > 2 {
		page, _ = strconv.Atoi(parts[2])
	}

	module, exists := GetModule(moduleName)
	modHelpText := "❌ No help found for this module."
	if exists && module.Help != "" {
		modHelpText = module.Help
	}

	backData := "help"
	if page > 0 {
		backData = fmt.Sprintf("help_page:%d", page)
	}

	keyboard := telegram.NewKeyboard().AddRow(
		telegram.Button.Data("⬅️ Back", backData),
	).Build()

	_, err := c.Edit(modHelpText, &telegram.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}
