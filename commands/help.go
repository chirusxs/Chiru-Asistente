package commands

import (
	"fmt"
	"strings"

	"github.com/botlabs-gg/yagpdb/v2/bot/paginatedmessages"
	"github.com/botlabs-gg/yagpdb/v2/common"
	"github.com/botlabs-gg/yagpdb/v2/lib/dcmd"
	"github.com/botlabs-gg/yagpdb/v2/lib/discordgo"
	"github.com/botlabs-gg/yagpdb/v2/web"
)

var cmdHelp = &YAGCommand{
	Name:        "Ayuda",
	Aliases:     []string{"commands", "h", "how", "command"},
	Description: "Shows help about all or one specific command",
	CmdCategory: CategoryGeneral,
	RunInDM:     true,
	Arguments: []*dcmd.ArgDef{
		{Name: "command", Type: dcmd.String},
	},

	RunFunc:  cmdFuncHelp,
	Cooldown: 10,
}

func CmdNotFound(search string) string {
	return fmt.Sprintf("No se pudo encontrar el comando '%s'", search)
}

func cmdFuncHelp(data *dcmd.Data) (interface{}, error) {
	target := data.Args[0].Str()

	// Send the targetted help in the channel it was requested in
	resp := dcmd.GenerateTargettedHelp(target, data, data.ContainerChain[0], &dcmd.StdHelpFormatter{})
	for _, v := range resp {
		ensureEmbedLimits(v)
	}

	if target != "" {
		if len(resp) != 1 {
			// Send command not found in same channel
			return CmdNotFound(target), nil
		}

		// see if we can find the permissions the command needs and add that info to the help message
		cmd, _ := data.ContainerChain[0].AbsFindCommand(target)
		if cmd == nil {
			return resp, nil
		}

		yc, ok := cmd.Command.(*YAGCommand)
		if !ok {
			return resp, nil
		}

		if len(yc.RequireDiscordPerms) == 0 && yc.RequiredDiscordPermsHelp == "" {
			return resp, nil
		}

		requiredPerms := yc.RequiredDiscordPermsHelp
		if requiredPerms == "" {
			humanizedPerms := make([]string, 0, len(yc.RequireDiscordPerms))
			for _, v := range yc.RequireDiscordPerms {
				h := common.HumanizePermissions(v)
				if len(h) == 1 {
					humanizedPerms = append(humanizedPerms, h[0])
				} else {
					joined := strings.Join(h, " y ")
					humanizedPerms = append(humanizedPerms, "("+joined+")")
				}
			}
			requiredPerms = strings.Join(humanizedPerms, " o ")
		}

		embed := resp[0]
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: "Permiso(s) requerido(s): " + requiredPerms,
		}
		return embed, nil
	}

	// Send full help in DM
	ir, err := createInteractiveHelp(data.Author.ID, resp)
	if ir != nil || err != nil {
		return ir, err
	}

	if data.Source == dcmd.TriggerSourceDM {
		return nil, nil
	}

	return "📫 ¡Revisa tus mensajes privados!", nil
}

func createInteractiveHelp(userID int64, helpEmbeds []*discordgo.MessageEmbed) (interface{}, error) {
	channel, err := common.BotSession.UserChannelCreate(userID)
	if err != nil {
		return "Algo salió mal. Revisa que el bot pueda enviarte mensajes privados.", err
	}

	// prepend a introductionairy first page
	firstPage := &discordgo.MessageEmbed{
		Title: "Chiru Asistente - Ayuda",
		Color: 0xFF498D,
		Description: fmt.Sprintf(`Chiru Asistente es un bot privado de la comunidad CHIRUSXS.
Puedes visitar su sitio web haciendo [clic aquí](%s).
Si necesitas ayuda, abre un ticket en el canal <#832033137742708737>
		
**¡Cambia de páginas con los botones de debajo!**`, web.BaseURL()),
	}

	var pageLayout strings.Builder
	for i, v := range helpEmbeds {
		pageLayout.WriteString(fmt.Sprintf("**Página %d**: %s\n", i+2, v.Title))
	}
	firstPage.Fields = []*discordgo.MessageEmbedField{
		{Name: "Páginas de ayuda", Value: pageLayout.String()},
	}

	helpEmbeds = append([]*discordgo.MessageEmbed{firstPage}, helpEmbeds...)

	_, err = paginatedmessages.CreatePaginatedMessage(0, channel.ID, 1, len(helpEmbeds), func(p *paginatedmessages.PaginatedMessage, page int) (*discordgo.MessageEmbed, error) {
		embed := helpEmbeds[page-1]
		return embed, nil
	})
	if err != nil {
		return "Algo salió mal. Revisa que el bot pueda enviarte mensajes privados.", err

	}

	return nil, nil
}
