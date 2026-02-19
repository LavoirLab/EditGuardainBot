package modules

import "github.com/amarnathcjd/gogram/telegram"

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "callback",
		ModName: "Callback",
		Help:    "",
		Handlers: []HandlerConfig{
			{
				Pattern: "callback:close",
				Handler: closeCB,
			},
		},
	})
}

func closeCB(cq *telegram.CallbackQuery) error {
	cq.Answer("")
	cq.Delete()
	return telegram.ErrEndGroup
}
