package interactive

import (
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/util"
)

func Continue(action string) bool {
	return Confirm("Continue "+action+"?", true)
}

func ConfirmOverwrite(path string) bool {
	return Confirm("Overwrite "+path+"?", false)
}

func Confirm(message string, def bool) bool {
	var cont = &survey.Confirm{
		Message: message,
		Default: def,
	}

	var contResult bool
	err := survey.AskOne(cont, &contResult)
	if err != nil {
		return false
	}

	return contResult
}

func getFixedArray(config *ini.ServerConfig, key string) []string {
	list := strings.Split(config.GetOrDefault(key, ""), ";")
	fixed := make([]string, 0)
	for _, id := range list {
		if id == "" {
			continue
		}

		fixed = append(fixed, fixSeparator(id)...)
	}

	return util.Dedupe(fixed)
}

func fixSeparator(id string) []string {
	ids := strings.Split(id, ",")
	for i, id := range ids {
		ids[i] = strings.TrimSpace(id)
	}

	return ids
}

func isEnabled(id string, list []string) bool {
	for _, mod := range list {
		if mod == id {
			return true
		}
	}

	return false
}
