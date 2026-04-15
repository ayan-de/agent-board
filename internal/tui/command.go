package tui

import "strings"

type Item struct {
	Label       string
	Description string
	ID          string
}

type Command struct {
	Name        string
	Description string
	Prefix      string
	Items       func() []Item
}

type CommandRegistry struct {
	commands []Command
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{}
}

func (cr *CommandRegistry) Register(cmd Command) {
	cr.commands = append(cr.commands, cmd)
}

func (cr *CommandRegistry) All() []Command {
	return cr.commands
}

func (cr *CommandRegistry) Filter(query string) []Command {
	if !strings.HasPrefix(query, "/") {
		return cr.commands
	}
	name := strings.TrimPrefix(query, "/")
	var filtered []Command
	for _, cmd := range cr.commands {
		if strings.HasPrefix(cmd.Name, name) || strings.Contains(strings.ToLower(cmd.Name), strings.ToLower(name)) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}
