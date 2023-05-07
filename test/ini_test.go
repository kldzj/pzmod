package test

import (
	"os"
	"testing"

	"github.com/kldzj/pzmod/ini"
)

const testIniPath = "servertest.ini"
const testIni2Path = "servertest2.ini"

func TestLoadIni(t *testing.T) {
	config, _ := ini.NewServerConfig(testIniPath)
	config.Load()

	if len(config.Keys) == 0 {
		t.Errorf("config.Keys is empty")
	}

	name, exists := config.Get("PublicName")
	if !exists {
		t.Errorf("config.Get(\"PublicName\") failed")
	}

	if name != "pzmod" {
		t.Errorf("config.Get(\"PublicName\") returned %s, expected %s", name, "pzmod")
	}

	maxPlayers, exists := config.Get("MaxPlayers")
	if !exists {
		t.Errorf("config.Get(\"MaxPlayers\") failed")
	}

	if maxPlayers != "32" {
		t.Errorf("config.Get(\"MaxPlayers\") returned %s, expected %s", maxPlayers, "32")
	}

	mods, exists := config.Get("Mods")
	if !exists {
		t.Errorf("config.Get(\"Mods\") failed")
	}

	if mods != "" {
		t.Errorf("config.Get(\"Mods\") returned %s, expected an empty string", mods)
	}
}

func TestSaveIni(t *testing.T) {
	config, _ := ini.NewServerConfig(testIniPath)
	config.Load()
	defer os.Remove(testIni2Path)
	config.SaveTo(testIni2Path)

	config2, _ := ini.NewServerConfig(testIni2Path)
	config2.Load()

	if config.String() != config2.String() {
		t.Errorf("config.String() != config2.String()")
	}
}

func TestIniIntegrity(t *testing.T) {
	config, err := os.ReadFile(testIniPath)
	if err != nil {
		t.Errorf("os.ReadFile(testIniPath) failed")
	}

	config2, _ := ini.NewServerConfig(testIni2Path)
	config2.FromString(string(config))

	if config2.String() != string(config) {
		t.Errorf("config2.String() != string(config)")
	}
}
