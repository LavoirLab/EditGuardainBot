// file: modules/modules.go - UPDATED VERSION
package modules

import (
	"fmt"
	"log"
	"slices"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
)

type HelpModule struct {
	Callback string
	Help     string
}

type DeferredHandler struct {
	Pattern any
	Handler any
	Filters []telegram.Filter
}

var (
	Continue error = nil
	Commands       = []string{
		"/biolink",
		"/setlonglimit",
		"/setlongmode",
		"/echo",
		"/gcast",
		"/ping",
		"/reload",
		"/start",
		"/stats",
		"/help",
		"/nolinks",
		"/noabuse",
		"/allowlink",
		"/allowhost",
		"/removelink",
		"/listlinks",
		"/settings",
		"/setting",
		// NEW COMMANDS
		"/mediadelete",
		"/setmediadelay",
		"/mediaexcept",
		"/filedelete",
		"/fileexcept",
		"/msgdelete",
		"/setmsgdelay",
		"/noforward",
		"/nophone",
		"/purge",
		"/nohashtags",
		"/nopromo",
	}
)
var commandSet map[string]struct{}
var (
	ModulesHelp = make(map[string]*HelpModule, 0)
	handler     = make([]DeferredHandler, 0)
)
var BotInfo *telegram.UserObj

func init() {
	commandSet = make(map[string]struct{}, len(Commands))
	for _, cmd := range Commands {
		commandSet[cmd] = struct{}{}
	}
}

func FilterOwner(m *telegram.NewMessage) bool {
	return slices.Contains(config.OwnerId, m.SenderID())
}

func LoadMods(c *telegram.Client) {
	c.UpdatesGetState()

	// Existing commands
	c.On("command:biolink", setBioMode)
	c.On("command:setlonglimit", SetLongLimitHandler)
	c.On("command:gcast", BroadcastFunc)
	c.On("command:setlongmode", SetLongModeHandler)
	c.On("command:echo", EcoHandler)
	c.On("command:ping", pingHandler)
	c.On("command:reload", ReloadHandler)
	c.On("command:start", start)
	c.On("command:stats", stats)
	c.On("command:help", help)
	c.On("command:nolinks", NoLinksCmd)
	c.On("command:noabuse", NoAbuseCmd)
	c.On("command:allowlink", AllowHostCmd)
	c.On("command:allowhost", AllowHostCmd)
	c.On("command:removelink", RemoveHostCmd)
	c.On("command:listlinks", ListAllowedHosts)

	// NEW FEATURE COMMANDS
	c.On("command:mediadelete", MediaDeleteCmd)
	c.On("command:setmediadelay", SetMediaDelayCmd)
	c.On("command:mediaexcept", MediaExceptCmd)
	c.On("command:msgdelete", MsgDeleteCmd)
	c.On("command:setmsgdelay", SetMsgDelayCmd)
	c.On("command:noforward", NoForwardCmd)
	c.On("command:nophone", NoPhoneCmd)
	c.On("command:purge", PurgeCmd)
	c.On("command:(nohashtags|nohashtag)", NoHashtagsCmd)
	c.On("command:(nopromo|nopromotion)", NoPromoCmd)

	// Owner commands
	c.On("command:sh", ShellHandle, telegram.FilterFunc(FilterOwner))
	c.On("command:bash", ShellHandle, telegram.FilterFunc(FilterOwner))
	c.On("command:ls", LsHandler, telegram.FilterFunc(FilterOwner))
	c.On("command:eval", EvalHandle, telegram.FilterFunc(FilterOwner))

	// Message handlers
	c.On("message", OnMessageFnc)
	c.On("edit", deleteEditedMessage)
	c.On("participant", botAddded)

	// Callback handlers
	c.On("callback:close", closeCB)
	c.On("callback:help", helpCB)
	c.On("callback:start_callback", startCB)

	for _, h := range handler {
		c.On(h.Pattern, h.Handler, h.Filters...)
	}
}

func AddHelp(name, callback, help string, filters ...telegram.Filter) {
	handler = append(handler, DeferredHandler{
		Pattern: "callback:" + callback,
		Handler: helpModuleCB,
		Filters: filters,
	})
	ModulesHelp[name] = &HelpModule{
		Callback: callback,
		Help:     help,
	}
}

func GetHelp(callback string) string {
	for _, data := range ModulesHelp {
		if data.Callback == callback {
			return data.Help
		}
	}
	return ""
}

func orCont(err error) error {
	if err != nil {
		return err
	}
	return telegram.EndGroup
}

func L(m *telegram.NewMessage, context string, err error) error {
	if err == nil {
		return telegram.EndGroup
	}
	if BotInfo == nil {
		me, meErr := m.Client.GetMe()
		if meErr != nil {
			return telegram.EndGroup
		}
		BotInfo = me
	}
	msg := fmt.Sprintf(
		"<b>⚠️ Error Occurred</b>\n"+
			"<b>🔹 Context:</b> <code>%s</code>\n"+
			"%s"+
			"<b>🗨️ Message:</b> <code>%s</code>\n"+
			"<b>❗ Error:</b> <code>%s</code>",
		context,
		func() string {
			if m.GetCommand() != "" {
				return fmt.Sprintf("<b>💬 Command:</b> <code>%s</code>\n", m.Text())
			}
			return ""
		}(),
		m.Text(),
		err.Error(),
	)

	if BotInfo.Username != "ViyomBot" && BotInfo.Username != "MasterGuardiansBot" {
		m.Client.SendMessage("@Viyomx", msg)
		return telegram.EndGroup
	}

	log.Printf("[ERROR] %s: %v", context, err)

	m.Client.SendMessage(config.LoggerId, msg)
	for id := range config.OwnerId {
		m.Client.SendMessage(id, msg)
	}

	return telegram.EndGroup
}
