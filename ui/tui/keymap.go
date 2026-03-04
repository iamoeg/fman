package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap holds keybindings that are always active.
type GlobalKeyMap struct {
	SwitchPane key.Binding
	Quit       key.Binding
}

var globalKeys = GlobalKeyMap{
	SwitchPane: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "switch pane"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// SidebarKeyMap holds keybindings active when the sidebar is focused.
type SidebarKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
}

var sidebarKeys = SidebarKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
}

// MainKeyMap holds keybindings active in the main pane (no modal).
type MainKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	New    key.Binding
	Edit   key.Binding
	Delete key.Binding
	Filter key.Binding
	Back   key.Binding
}

var mainKeys = MainKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j", "down"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
}
