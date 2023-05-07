package interactive

import (
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdUpdateServerInfo(cmd *cobra.Command, config *ini.ServerConfig) {
	qs := []*survey.Question{
		{
			Name: "name",
			Prompt: &survey.Input{
				Message: "Server name:",
				Default: config.GetOrDefault(util.CfgKeyName, "pzmod"),
			},
		},
		{
			Name: "description",
			Prompt: &survey.Multiline{
				Message: "Server description:",
				Default: config.GetOrDefault(util.CfgKeyDesc, ""),
			},
		},
		{
			Name: "public",
			Prompt: &survey.Confirm{
				Message: "Public server?",
				Default: config.GetOrDefault(util.CfgKeyPub, "true") == "true",
			},
		},
		{
			Name: "password",
			Prompt: &survey.Input{
				Message: "Server password:",
				Default: config.GetOrDefault(util.CfgKeyPass, ""),
			},
		},
		{
			Name: "maxplayers",
			Prompt: &survey.Input{
				Message: "Max players:",
				Default: config.GetOrDefault(util.CfgKeyMax, "8"),
			},
		},
	}

	answers := struct {
		Name        string
		Description string
		Public      bool
		Password    string
		MaxPlayers  string `survey:"maxplayers"`
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		cmd.Println(util.Error, err)
		cmd.Println(util.Warning, "Server info not updated")
		return
	}

	config.Set(util.CfgKeyName, answers.Name)
	config.Set(util.CfgKeyDesc, strings.Join(strings.Split(strings.TrimSpace(answers.Description), "\n"), "<line>"))
	config.Set(util.CfgKeyPub, util.BoolString(answers.Public))
	config.Set(util.CfgKeyPass, answers.Password)
	config.Set(util.CfgKeyMax, answers.MaxPlayers)
	cmd.Println(util.OK, "Updated server info")
}
