package modprobe

import (
	"fmt"
	"strings"
)

// Config represents a modprobe.d config file.
type Config struct {
	lines []string
}

// Parse parses modprobe config content.
func Parse(content string) *Config {
	return &Config{lines: strings.Split(content, "\n")}
}

// Aliases returns all alias directives.
func (c *Config) Aliases() map[string]string {
	result := make(map[string]string)
	for _, line := range c.lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "alias ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				result[parts[1]] = parts[2]
			}
		}
	}
	return result
}

// AddAlias adds a module alias. Replaces if the alias pattern already exists.
func (c *Config) AddAlias(pattern, module string) {
	line := fmt.Sprintf("alias %s %s", pattern, module)
	for i, l := range c.lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "alias "+pattern+" ") {
			c.lines[i] = line
			return
		}
	}
	c.lines = append(c.lines, line)
}

// AddOption adds a module option line.
func (c *Config) AddOption(module, options string) {
	c.lines = append(c.lines, fmt.Sprintf("options %s %s", module, options))
}

// AddBlacklist adds a blacklist directive.
func (c *Config) AddBlacklist(module string) {
	c.lines = append(c.lines, fmt.Sprintf("blacklist %s", module))
}

// String serializes back.
func (c *Config) String() string {
	return strings.Join(c.lines, "\n")
}
