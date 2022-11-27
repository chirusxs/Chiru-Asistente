package customembed

import (
	"encoding/json"

	"github.com/botlabs-gg/yagpdb/v2/commands"
	"github.com/botlabs-gg/yagpdb/v2/lib/dcmd"
	"github.com/botlabs-gg/yagpdb/v2/lib/discordgo"
)

var Command = &commands.YAGCommand{
	CmdCategory:         commands.CategoryTool,
	Name:                "customembed",
	Description:         "Crea un embed con formato json",
	LongDescription:     "Ejemplos: `-ce {\"title\": \"¡Esto es un título!\", \"description\": \"Qué linda descripción.\"}`",
	RequiredArgs:        1,
	RequireDiscordPerms: []int64{discordgo.PermissionManageMessages},
	Arguments: []*dcmd.ArgDef{
		{Name: "Json", Type: dcmd.String},
	},
	RunFunc: func(data *dcmd.Data) (interface{}, error) {
		var parsed *discordgo.MessageEmbed
		err := json.Unmarshal([]byte(data.Args[0].Str()), &parsed)
		if err != nil {
			return "Failed parsing json: " + err.Error(), err
		}
		if discordgo.IsEmbedEmpty(parsed) {
			return "Cannot send an empty embed", nil
		}
		return parsed, nil
	},
}
