package automod_legacy

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"

	"github.com/botlabs-gg/yagpdb/v2/common/cplogs"
	"github.com/botlabs-gg/yagpdb/v2/common/featureflags"
	"github.com/botlabs-gg/yagpdb/v2/common/pubsub"
	"github.com/botlabs-gg/yagpdb/v2/lib/discordgo"
	"github.com/botlabs-gg/yagpdb/v2/web"
	"goji.io"
	"goji.io/pat"
)

//go:embed assets/automod_legacy.html
var PageHTML string

type GeneralForm struct {
	Enabled bool
}

var panelLogKeyUpdatedSettings = cplogs.RegisterActionFormat(&cplogs.ActionFormat{Key: "automod_legacy_settings_updated", FormatString: "Se cambiaron los ajustes de la automoderación básica"})

func (p *Plugin) InitWeb() {
	web.AddHTMLTemplate("automod_legacy/assets/automod_legacy.html", PageHTML)

	web.AddSidebarItem(web.SidebarCategoryTools, &web.SidebarItem{
		Name: "Automoderación básica",
		URL:  "automod_legacy",
		Icon: "fas fa-robot",
	})

	autmodMux := goji.SubMux()
	web.CPMux.Handle(pat.New("/automod_legacy/*"), autmodMux)
	web.CPMux.Handle(pat.New("/automod_legacy"), autmodMux)

	// All handlers here require guild channels present
	autmodMux.Use(web.RequireBotMemberMW)
	autmodMux.Use(web.RequirePermMW(discordgo.PermissionManageRoles, discordgo.PermissionKickMembers, discordgo.PermissionBanMembers, discordgo.PermissionManageMessages, discordgo.PermissionManageServer, discordgo.PermissionModerateMembers))

	getHandler := web.RenderHandler(HandleAutomod, "cp_automod_legacy")

	autmodMux.Handle(pat.Get("/"), getHandler)
	autmodMux.Handle(pat.Get(""), getHandler)

	// Post handlers
	autmodMux.Handle(pat.Post("/"), ExtraPostMW(web.SimpleConfigSaverHandler(Config{}, getHandler, panelLogKeyUpdatedSettings)))
	autmodMux.Handle(pat.Post(""), ExtraPostMW(web.SimpleConfigSaverHandler(Config{}, getHandler, panelLogKeyUpdatedSettings)))
}

func HandleAutomod(w http.ResponseWriter, r *http.Request) interface{} {
	g, templateData := web.GetBaseCPContextData(r.Context())

	config, err := GetConfig(g.ID)
	web.CheckErr(templateData, err, "Failed retrieving rules", web.CtxLogger(r.Context()).Error)

	templateData["AutomodConfig"] = config
	templateData["VisibleURL"] = "/manage/" + discordgo.StrID(g.ID) + "/automod_legacy/"

	return templateData
}

// Invalidates the cache
func ExtraPostMW(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		activeGuild, _ := web.GetBaseCPContextData(r.Context())
		pubsub.Publish("update_automod_legacy_rules", activeGuild.ID, nil)
		featureflags.MarkGuildDirty(activeGuild.ID)
		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}

var _ web.PluginWithServerHomeWidget = (*Plugin)(nil)

func (p *Plugin) LoadServerHomeWidget(w http.ResponseWriter, r *http.Request) (web.TemplateData, error) {
	g, templateData := web.GetBaseCPContextData(r.Context())

	templateData["WidgetTitle"] = "Automoderación básica"
	templateData["SettingsPath"] = "/automod_legacy"

	config, err := GetConfig(g.ID)
	if err != nil {
		return templateData, err
	}

	if config.Enabled {
		templateData["WidgetEnabled"] = true
	} else {
		templateData["WidgetDisabled"] = true
	}

	const format = `<ul>
	<li>Modo lento: %s</li>
	<li>Menciones masivas: %s</li>
	<li>Invitaciones a servidores: %s</li>
	<li>Enlaces: %s</li>
	<li>Palabras baneadas: %s</li>
	<li>Sitios web baneados: %s</li>
</ul>`

	slowmode := web.EnabledDisabledSpanStatus(config.Spam.Enabled)
	massMention := web.EnabledDisabledSpanStatus(config.Mention.Enabled)
	invites := web.EnabledDisabledSpanStatus(config.Invite.Enabled)
	links := web.EnabledDisabledSpanStatus(config.Links.Enabled)
	words := web.EnabledDisabledSpanStatus(config.Words.Enabled)
	sites := web.EnabledDisabledSpanStatus(config.Sites.Enabled)

	templateData["WidgetBody"] = template.HTML(fmt.Sprintf(format, slowmode,
		massMention, invites, links, words, sites))

	return templateData, nil
}
