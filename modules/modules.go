package modules

import (
	"errors"
	"fmt"
	"html"
	"log"
	"runtime/debug"
	"sort"
	"strconv"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/utils"
)

type ModuleInfo struct {
	Name     string
	ModName  string
	Help     string
	Handlers []HandlerConfig
}

type HandlerConfig struct {
	Pattern string
	Handler any
	Group   int
}

var registeredModules = make(map[string]*ModuleInfo)
var BotInfo *tg.UserObj

func RegisterModule(info *ModuleInfo) {
	if _, exists := registeredModules[info.Name]; exists {
		log.Fatalf("Module '%s' is already registered", info.Name)
	}

	registeredModules[info.Name] = info
	log.Printf("✓ Registered module: %s", info.ModName)
}

func GetModule(name string) (*ModuleInfo, bool) {
	mod, exists := registeredModules[name]
	return mod, exists
}

func GetAllModules() []*ModuleInfo {
	modules := make([]*ModuleInfo, 0, len(registeredModules))
	for _, mod := range registeredModules {
		modules = append(modules, mod)
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].ModName < modules[j].ModName
	})

	return modules
}

func GetModulesWithHelp() []*ModuleInfo {
	modules := make([]*ModuleInfo, 0)
	for _, mod := range registeredModules {
		if mod.Help != "" {
			modules = append(modules, mod)
		}
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].ModName < modules[j].ModName
	})

	return modules
}

func LoadAllHandlers(client *tg.Client) {
	log.Println("Loading handlers from registered modules...")

	names := make([]string, 0, len(registeredModules))
	for name := range registeredModules {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		module := registeredModules[name]

		for _, hConfig := range module.Handlers {
			group := hConfig.Group
			if group == 0 {
				group = 10
			}

			var handle tg.Handle

			switch h := hConfig.Handler.(type) {
			case func(*tg.NewMessage) error:
				wrapped := safeMessageHandler(h)
				handle = client.On(hConfig.Pattern, wrapped)

			case func(*tg.CallbackQuery) error:
				wrapped := safeCallbackHandler(h)
				handle = client.On(hConfig.Pattern, wrapped)
			default:
				handle = client.On(hConfig.Pattern, h)
			}

			if handle != nil {
				handle.SetGroup(strconv.Itoa(group))
			}
		}

		log.Printf("✓ Loaded handlers for module: %s", module.ModName)
	}

	log.Printf("✓ Successfully loaded %d modules", len(registeredModules))
}

func safeCallbackHandler(
	handler func(*tg.CallbackQuery) error,
) func(*tg.CallbackQuery) error {
	return func(cb *tg.CallbackQuery) (err error) {
		defer func() {
			if r := recover(); r != nil {
				handlePanic(r, cb, true)
				err = tg.EndGroup
			}
		}()
		err = handler(cb)
		if err != nil {
			if errors.Is(err, tg.EndGroup) {
				return err
			}
			handlePanic(err, cb, false)
		}
		return tg.EndGroup
	}
}

func safeMessageHandler(
	handler func(*tg.NewMessage) error,
) func(*tg.NewMessage) error {
	return func(m *tg.NewMessage) (err error) {
		if m.Message != nil && m.Message.Post {
			return nil
		}

		m.Client.Log.Info(
			fmt.Sprintf("Handling message from %d in chat %d",
				m.SenderID(),
				m.ChannelID(),
			),
		)

		defer func() {
			if r := recover(); r != nil {
				handlePanic(r, m, true)
				err = tg.EndGroup
			}
		}()

		cmd := m.GetCommand()
		if cmd != "" {
			m.Client.Log.Debug("Executing handler for command " + cmd)
		}
		err = handler(m)

		if err != nil {
			if errors.Is(err, tg.EndGroup) {
				return err
			}
			m.Client.Log.Error("Handler error: " + err.Error())
			handlePanic(err, m, false)
		} else {
			if cmd != "" {
				m.Client.Log.Info("Handler completed successfully for command " + cmd)
			}
		}

		return tg.EndGroup
	}
}

func handlePanic(r, ctx interface{}, isPanic bool) {
	stack := html.EscapeString(string(debug.Stack()))

	var userMention, handlerType, chatInfo, messageInfo, errorMessage string
	var client *tg.Client

	switch c := ctx.(type) {
	case *tg.NewMessage:
		userMention = utils.MentionHTML(c.Sender)
		handlerType = "message"
		chatInfo = fmt.Sprintf("ChatID: %d", c.ChannelID())
		messageInfo = fmt.Sprintf("Message: %s\nLink: %s", html.EscapeString(c.Text()), c.Link())
		errorMessage = html.EscapeString(fmt.Sprint(r))
		client = c.Client

	case *tg.CallbackQuery:
		userMention = utils.MentionHTML(c.Sender)
		handlerType = "callback"
		chatInfo = fmt.Sprintf("ChatID: %d", c.ChatID)
		messageInfo = fmt.Sprintf("Data: %s", html.EscapeString(c.DataString()))
		errorMessage = html.EscapeString(fmt.Sprint(r))
		client = c.Client
	}

	if client == nil {
		return
	}

	logMsg := "🚨 Error in %s handler:\nFrom: %s\n%s\n%s\nError: `%v`"
	shortMsg := "<b>Error in %s handler</b>\n<b>From:</b> %s\n%s\n%s\n<b>Error:</b>\n<code>%s</code>"

	if isPanic {
		logMsg = "⚠️ Panic recovered in %s handler:\nFrom: %s\n%s\n%s\nError: `%v`\nStack:\n%s"
		shortMsg = "<b>⚠️ Panic in %s handler</b>\n<b>From:</b> %s\n%s\n%s\n<b>Error:</b>\n<code>%s</code>\n<pre>%s</pre>"
	}

	if isPanic {
		client.Log.Error(fmt.Sprintf(
			logMsg,
			handlerType,
			userMention,
			chatInfo,
			messageInfo,
			r,
			stack,
		))
	} else {
		client.Log.Error(fmt.Sprintf(logMsg, handlerType, userMention, chatInfo, messageInfo, r))
	}

	var short string
	if isPanic {
		short = fmt.Sprintf(
			shortMsg,
			handlerType,
			userMention,
			chatInfo,
			messageInfo,
			errorMessage,
			stack,
		)
	} else {
		short = fmt.Sprintf(shortMsg, handlerType, userMention, chatInfo, messageInfo, errorMessage)
	}

	if _, sendErr := client.SendMessage(config.LoggerId, short, &tg.SendOptions{ParseMode: "HTML"}); sendErr != nil {
		client.Log.Error(fmt.Sprintf(
			"Failed to send error message to log chat: %v",
			sendErr,
		))
	}
}
