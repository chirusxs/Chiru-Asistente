package reminders

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/botlabs-gg/yagpdb/v2/bot"
	"github.com/botlabs-gg/yagpdb/v2/commands"
	"github.com/botlabs-gg/yagpdb/v2/common"
	"github.com/botlabs-gg/yagpdb/v2/common/scheduledevents2"
	seventsmodels "github.com/botlabs-gg/yagpdb/v2/common/scheduledevents2/models"
	"github.com/botlabs-gg/yagpdb/v2/lib/dcmd"
	"github.com/botlabs-gg/yagpdb/v2/lib/discordgo"
	"github.com/botlabs-gg/yagpdb/v2/lib/dstate"
	"github.com/jinzhu/gorm"
)

var logger = common.GetPluginLogger(&Plugin{})

var _ bot.BotInitHandler = (*Plugin)(nil)
var _ commands.CommandProvider = (*Plugin)(nil)

func (p *Plugin) AddCommands() {
	commands.AddRootCommands(p, cmds...)
}

func (p *Plugin) BotInit() {
	// scheduledevents.RegisterEventHandler("reminders_check_user", checkUserEvtHandlerLegacy)
	scheduledevents2.RegisterHandler("reminders_check_user", int64(0), checkUserScheduledEvent)
	scheduledevents2.RegisterLegacyMigrater("reminders_check_user", migrateLegacyScheduledEvents)
}

// Reminder management commands
var cmds = []*commands.YAGCommand{
	{
		CmdCategory:  commands.CategoryTool,
		Name:         "recordatorio",
		Description:  "Programa un recordatorio para el futuro",
		Aliases:      []string{"recordar", "recordarme"},
		RequiredArgs: 2,
		Arguments: []*dcmd.ArgDef{
			{Name: "Time", Type: &commands.DurationArg{}},
			{Name: "Message", Type: dcmd.String},
		},
		ArgSwitches: []*dcmd.ArgDef{
			{Name: "channel", Type: dcmd.Channel},
		},
		SlashCommandEnabled: true,
		DefaultEnabled:      true,
		RunFunc: func(parsed *dcmd.Data) (interface{}, error) {
			currentReminders, _ := GetUserReminders(parsed.Author.ID)
			if len(currentReminders) >= 25 {
				return "¡Has alcanzado tu límite de 25 recordatorios simultáneos!", nil
			}

			fromNow := parsed.Args[0].Value.(time.Duration)

			durString := common.HumanizeDuration(common.DurationPrecisionSeconds, fromNow)
			when := time.Now().Add(fromNow)
			tUnix := fmt.Sprint(when.Unix())

			if when.After(time.Now().Add(time.Hour * 24 * 366)) {
				return "Can be max 365 days from now...", nil
			}

			id := parsed.ChannelID
			if c := parsed.Switch("channel"); c.Value != nil {
				id = c.Value.(*dstate.ChannelState).ID

				hasPerms, err := bot.AdminOrPermMS(parsed.GuildData.GS.ID, id, parsed.GuildData.MS, discordgo.PermissionSendMessages|discordgo.PermissionReadMessages)
				if err != nil {
					return "Ocurrió un error desconocido... :(", err
				}

				if !hasPerms {
					return "No tienes permisos para enviar mensajes en ese canal.", nil
				}
			}

			_, err := NewReminder(parsed.Author.ID, parsed.GuildData.GS.ID, id, parsed.Args[1].Str(), when)
			if err != nil {
				return nil, err
			}

			return "¡Estableciste un recordatorio en " + durString + " a partir de ahora! (<t:" + tUnix + ":f>)\nPuedes revisar tus recordatorios con el comando `recordatorios`", nil
		},
	},
	{
		CmdCategory:         commands.CategoryTool,
		Name:                "recordatorios",
		Description:         "Muestra tus recordatorios activos",
		SlashCommandEnabled: true,
		DefaultEnabled:      true,
		RunFunc: func(parsed *dcmd.Data) (interface{}, error) {
			currentReminders, err := GetUserReminders(parsed.Author.ID)
			if err != nil {
				return nil, err
			}

			out := "Tus recordatorios:\n"
			out += stringReminders(currentReminders, false)
			out += "\nPuedes eliminar un recordatorio con el comando `recordatorioeliminar (id)`.\nPara eliminar todos los recordatorios, utiliza `recordatorioeliminar -a`."
			return out, nil
		},
	},
	{
		CmdCategory:         commands.CategoryTool,
		Name:                "revisarrecordatorios",
		Description:         "Mostrar recordatorios en un canal.",
		SlashCommandEnabled: true,
		DefaultEnabled:      true,
		RunFunc: func(parsed *dcmd.Data) (interface{}, error) {
			ok, err := bot.AdminOrPermMS(parsed.GuildData.GS.ID, parsed.ChannelID, parsed.GuildData.MS, discordgo.PermissionManageChannels)
			if err != nil {
				return nil, err
			}
			if !ok {
				return "¡No tienes acceso a ese comando!", nil
			}

			currentReminders, err := GetChannelReminders(parsed.ChannelID)
			if err != nil {
				return nil, err
			}

			out := "Recordatorios en este canal:\n"
			out += stringReminders(currentReminders, true)
			out += "\nPuedes eliminar un recordatorio usando `recordatorioeliminar (id)`"
			return out, nil
		},
	},
	{
		CmdCategory:  commands.CategoryTool,
		Name:         "recordatorioeliminar",
		Description:  "Comando para eliminar recordatorios.",
		RequiredArgs: 0,
		Arguments: []*dcmd.ArgDef{
			{Name: "ID", Type: dcmd.Int},
		},
		ArgSwitches: []*dcmd.ArgDef{
			{Name: "a", Help: "All"},
		},
		SlashCommandEnabled: true,
		DefaultEnabled:      true,
		RunFunc: func(parsed *dcmd.Data) (interface{}, error) {
			var reminder Reminder

			clearAll := parsed.Switch("a").Value != nil && parsed.Switch("a").Value.(bool)
			if clearAll {
				db := common.GORM.Where("user_id = ?", parsed.Author.ID).Delete(&reminder)
				err := db.Error
				if err != nil {
					return "Ocurrió un error desconocido... :(", err
				}

				count := db.RowsAffected
				if count == 0 {
					return "No hay recordatorios para eliminar.", nil
				}
				return fmt.Sprintf("¡Se eliminaron %d recordatorios!", count), nil
			}

			if len(parsed.Args) == 0 || parsed.Args[0].Value == nil {
				return "¡Introduce una Id. de recordatorio válida!", nil
			}

			err := common.GORM.Where(parsed.Args[0].Int()).First(&reminder).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return "No se encontró un recordatorio con esa Id.", nil
				}
				return "Ocurrió un error desconocido... :(", err
			}

			// Check perms
			if reminder.UserID != discordgo.StrID(parsed.Author.ID) {
				if reminder.GuildID != parsed.GuildData.GS.ID {
					return "¡No puedes eliminar ese recordatorio!", nil
				}
				ok, err := bot.AdminOrPermMS(reminder.GuildID, reminder.ChannelIDInt(), parsed.GuildData.MS, discordgo.PermissionManageChannels)
				if err != nil {
					return nil, err
				}
				if !ok {
					return "¡No puedes eliminar un recordatorio creado por alguien más!", nil
				}
			}

			// Do the actual deletion
			err = common.GORM.Delete(reminder).Error
			if err != nil {
				return nil, err
			}

			// Check if we should remove the scheduled event
			currentReminders, err := GetUserReminders(reminder.UserIDInt())
			if err != nil {
				return nil, err
			}

			delMsg := fmt.Sprintf("Se eliminaron **#%d** recordatorios: '%s'", reminder.ID, limitString(reminder.Message))

			// If there is another reminder with the same timestamp, do not remove the scheduled event
			for _, v := range currentReminders {
				if v.When == reminder.When {
					return delMsg, nil
				}
			}

			return delMsg, nil
		},
	},
}

func stringReminders(reminders []*Reminder, displayUsernames bool) string {
	out := ""
	for _, v := range reminders {
		parsedCID, _ := strconv.ParseInt(v.ChannelID, 10, 64)

		t := time.Unix(v.When, 0)
		tUnix := t.Unix()
		timeFromNow := common.HumanizeTime(common.DurationPrecisionMinutes, t)
		if !displayUsernames {
			channel := "<#" + discordgo.StrID(parsedCID) + ">"
			out += fmt.Sprintf("**%d**: %s: '%s' - %s a partir de ahora (<t:%d:f>)\n", v.ID, channel, limitString(v.Message), timeFromNow, tUnix)
		} else {
			member, _ := bot.GetMember(v.GuildID, v.UserIDInt())
			username := "Miembro desconocido/a"
			if member != nil {
				username = member.User.Username
			}
			out += fmt.Sprintf("**%d**: %s: '%s' - %s a partir de ahora (<t:%d:f>)\n", v.ID, username, limitString(v.Message), timeFromNow, tUnix)
		}
	}
	return out
}

func checkUserScheduledEvent(evt *seventsmodels.ScheduledEvent, data interface{}) (retry bool, err error) {
	// !important! the evt.GuildID can be 1 in cases where it was migrated from the legacy scheduled event system

	userID := *data.(*int64)

	reminders, err := GetUserReminders(userID)
	if err != nil {
		return true, err
	}

	now := time.Now()
	nowUnix := now.Unix()
	for _, v := range reminders {
		if v.When <= nowUnix {
			err := v.Trigger()
			if err != nil {
				// possibly try again
				return scheduledevents2.CheckDiscordErrRetry(err), err
			}
		}
	}

	return false, nil
}

func migrateLegacyScheduledEvents(t time.Time, data string) error {
	split := strings.Split(data, ":")
	if len(split) < 2 {
		logger.Error("invalid check user scheduled event: ", data)
		return nil
	}

	parsed, _ := strconv.ParseInt(split[1], 10, 64)

	return scheduledevents2.ScheduleEvent("reminders_check_user", 1, t, parsed)
}

func limitString(s string) string {
	if utf8.RuneCountInString(s) < 50 {
		return s
	}

	runes := []rune(s)
	return string(runes[:47]) + "..."
}
