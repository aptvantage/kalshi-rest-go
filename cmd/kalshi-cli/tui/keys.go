package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap lists all global keybindings.
type KeyMap struct {
Up       key.Binding
Down     key.Binding
Enter    key.Binding
Back     key.Binding
Quit     key.Binding
Filter   key.Binding
Order    key.Binding
Tab      key.Binding
ShiftTab key.Binding
Space    key.Binding
CtrlS    key.Binding
}

// DefaultKeyMap is the application-wide keymap.
var DefaultKeyMap = KeyMap{
Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "select")),
Back:     key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
Order:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "new order")),
Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev field")),
Space:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
CtrlS:    key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "submit")),
}
