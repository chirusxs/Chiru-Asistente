package info

import (
	"fmt"

	"github.com/botlabs-gg/yagpdb/v2/commands"
	"github.com/botlabs-gg/yagpdb/v2/common"
	"github.com/botlabs-gg/yagpdb/v2/lib/dcmd"
)

var Command = &commands.YAGCommand{
	CmdCategory: commands.CategoryGeneral,
	Name:        "info",
	Description: "Muestra información del bot",
	RunInDM:     true,
	RunFunc: func(data *dcmd.Data) (interface{}, error) {
		info := fmt.Sprintf(`**Chiru Asistente - Instancia privada de YAGPDB**
Este bot es utilizado principalmente como herramienta de organización y moderación por el personal de la comunidad.
Sus sistemas ofrecen desde comandos de moderación hasta notificaciones de redes sociales.
Este bot es de código abierto y puedes encontrar su repositorio de GitHub en (<https://github.com/chirusxs/Chiru-Asistente>).
Sitio web (solo para ver información): <https://%s/manage>
				`, common.ConfHost.GetString())

		return info, nil
	},
}
