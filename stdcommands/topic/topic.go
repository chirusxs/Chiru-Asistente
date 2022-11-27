package topic

import (
	"math/rand"

	"github.com/botlabs-gg/yagpdb/v2/commands"
	"github.com/botlabs-gg/yagpdb/v2/lib/dcmd"
)

var Command = &commands.YAGCommand{
	Cooldown:            5,
	CmdCategory:         commands.CategoryFun,
	Name:                "tema",
	Description:         "Genera un tema de conversación",
	DefaultEnabled:      true,
	SlashCommandEnabled: true,
	RunFunc: func(data *dcmd.Data) (interface{}, error) {
		topic := ChatTopics[rand.Intn(len(ChatTopics))]
		return "> " + topic, nil
	},
}
