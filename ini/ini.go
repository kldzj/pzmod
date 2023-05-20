package ini

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kldzj/pzmod/eol"
)

type ConfigKey struct {
	Key      string
	Value    string
	Comments []string
}

type ServerConfig struct {
	Path string
	EOL  string
	Keys []ConfigKey
}

func NewServerConfig(configPath string) (*ServerConfig, error) {
	if !filepath.IsAbs(configPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}

		configPath = filepath.Join(cwd, configPath)
	}

	return &ServerConfig{Path: configPath, EOL: eol.OSDefault().String()}, nil
}

func LoadNewServerConfig(configPath string) (*ServerConfig, error) {
	c, err := NewServerConfig(configPath)
	if err != nil {
		return nil, err
	}

	err = c.Load()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *ServerConfig) String() string {
	lines := []string{}
	for _, configKey := range c.Keys {
		lines = append(lines, c.formatKey(configKey))
	}

	return strings.Join(lines, c.EOL)
}

func (c *ServerConfig) formatKey(configKey ConfigKey) string {
	lines := []string{}
	lines = append(lines, configKey.Comments...)
	lines = append(lines, fmt.Sprintf("%s=%s", configKey.Key, configKey.Value))
	return strings.Join(lines, c.EOL) + c.EOL
}

func (c *ServerConfig) FromString(data string) {
	c.EOL = eol.DetectDefault(data, eol.OSDefault()).String()
	lines := strings.Split(data, c.EOL)
	c.reset()
	c.parseLines(lines)
}

func (c *ServerConfig) Load() error {
	data, error := os.ReadFile(c.Path)
	if error != nil {
		return error
	}

	c.FromString(string(data))
	return nil
}

func (c *ServerConfig) Save() error {
	return c.SaveTo(c.Path)
}

func (c *ServerConfig) SaveTo(path string) error {
	err := os.WriteFile(path, []byte(c.String()), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (c *ServerConfig) HasUnsavedChanges() bool {
	stored, err := LoadNewServerConfig(c.Path)
	if err != nil {
		return true
	}

	return c.String() != stored.String()
}

func (c *ServerConfig) Get(key string) (string, bool) {
	for _, configKey := range c.Keys {
		if configKey.Key == key {
			return configKey.Value, true
		}
	}

	return "", false
}

func (c *ServerConfig) GetOrDefault(key string, defaultValue string) string {
	value, ok := c.Get(key)
	if !ok {
		return defaultValue
	}

	return value
}

func (c *ServerConfig) Set(key string, value string) {
	for i, configKey := range c.Keys {
		if configKey.Key == key {
			c.Keys[i].Value = value
			return
		}
	}

	c.addKey(key, value, []string{})
}

func (c *ServerConfig) reset() {
	c.Keys = []ConfigKey{}
}

func (c *ServerConfig) parseLines(lines []string) {
	comments := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, "#") {
			comments = append(comments, line)
			continue
		}

		if strings.Contains(line, "=") {
			c.parseLine(line, comments)
			comments = []string{}
		}
	}
}

func (c *ServerConfig) parseLine(line string, comments []string) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		fmt.Println("Invalid line: ", line)
		return
	}

	key := strings.TrimSpace(parts[0])
	valueParts := strings.SplitN(parts[1], "#", 2)
	value := strings.TrimSpace(valueParts[0])
	if len(valueParts) > 1 {
		comment := strings.TrimSpace(valueParts[1])
		comments = append(comments, comment)
	}

	c.addKey(key, value, comments)
}

func (c *ServerConfig) addKey(key string, value string, comments []string) {
	configKey := ConfigKey{Key: key, Value: value, Comments: comments}
	c.Keys = append(c.Keys, configKey)
}
