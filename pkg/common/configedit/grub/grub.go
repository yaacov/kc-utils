package grub

import (
	"strings"
)

// Config represents parsed GRUB2 /etc/default/grub settings.
type Config struct {
	lines []string
	kv    map[string]string
}

// Parse parses /etc/default/grub content.
func Parse(content string) *Config {
	c := &Config{
		kv: make(map[string]string),
	}
	c.lines = strings.Split(content, "\n")
	for _, line := range c.lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if idx := strings.Index(trimmed, "="); idx > 0 {
			key := trimmed[:idx]
			val := strings.Trim(trimmed[idx+1:], "\"'")
			c.kv[key] = val
		}
	}
	return c
}

// Get returns a config value.
func (c *Config) Get(key string) string {
	return c.kv[key]
}

// Set sets a config value, updating existing or appending.
func (c *Config) Set(key, value string) {
	c.kv[key] = value
	found := false
	for i, line := range c.lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			c.lines[i] = key + "=\"" + value + "\""
			found = true
			break
		}
	}
	if !found {
		c.lines = append(c.lines, key+"=\""+value+"\"")
	}
}

// GetKernelArgs returns the GRUB_CMDLINE_LINUX value split into args.
func (c *Config) GetKernelArgs() []string {
	val := c.kv["GRUB_CMDLINE_LINUX"]
	if val == "" {
		return nil
	}
	return strings.Fields(val)
}

// AddKernelArg adds an argument to GRUB_CMDLINE_LINUX if not present.
func (c *Config) AddKernelArg(arg string) {
	args := c.GetKernelArgs()
	for _, a := range args {
		if a == arg || strings.HasPrefix(a, strings.SplitN(arg, "=", 2)[0]+"=") {
			return
		}
	}
	args = append(args, arg)
	c.Set("GRUB_CMDLINE_LINUX", strings.Join(args, " "))
}

// RemoveKernelArg removes arguments matching the prefix from GRUB_CMDLINE_LINUX.
func (c *Config) RemoveKernelArg(prefix string) {
	args := c.GetKernelArgs()
	var filtered []string
	for _, a := range args {
		if a != prefix && !strings.HasPrefix(a, prefix+"=") {
			filtered = append(filtered, a)
		}
	}
	c.Set("GRUB_CMDLINE_LINUX", strings.Join(filtered, " "))
}

// String serializes back to /etc/default/grub format.
func (c *Config) String() string {
	return strings.Join(c.lines, "\n")
}
