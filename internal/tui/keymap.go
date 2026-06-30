package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the global key bindings. Screens may interpret additional keys.
type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding
	Help  key.Binding
	Save  key.Binding
}

// DefaultKeyMap returns the standard bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Back:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:  key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Help:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Save:  key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	}
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Back, k.Save},
		{k.Help, k.Quit},
	}
}
