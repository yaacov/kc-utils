//go:build linux

package guestfs

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func quoteGuestfish(path string) string {
	if strings.ContainsAny(path, " \t\"'") {
		return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
	}
	return path
}

// shellQuote quotes a string for safe use in a POSIX shell command line.
// It wraps the value in single quotes, escaping any embedded single
// quotes with the standard end-reopen idiom.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`!#&|;(){}[]<>?*~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func runGuestfsCmd(name string, args ...string) ([]byte, error) {
	slog.Info("guestfs exec", "bin", name, "args", args)
	start := time.Now()
	out, err := exec.Command(name, args...).CombinedOutput()
	duration := time.Since(start).Round(time.Millisecond)
	msg := strings.TrimSpace(string(out))
	if err != nil {
		slog.Error("guestfs exec failed", "bin", name, "duration", duration, "error", err, "output", msg)
		if msg != "" {
			return out, fmt.Errorf("%s: %w\n%s", name, err, msg)
		}
		return out, fmt.Errorf("%s: %w", name, err)
	}
	slog.Info("guestfs exec ok", "bin", name, "duration", duration, "outputBytes", len(out), "output", truncateLog(msg, 512))
	return out, nil
}

func runGuestfishScript(args []string, script string) ([]byte, error) {
	safe := prefixDash(script)
	slog.Info("guestfs exec",
		"bin", "guestfish",
		"args", args,
		"scriptLines", strings.Count(safe, "\n"),
		"script", truncateLog(strings.TrimSpace(safe), 256),
	)
	start := time.Now()
	cmd := exec.Command("guestfish", args...)
	cmd.Stdin = strings.NewReader(safe)
	out, err := cmd.CombinedOutput()
	duration := time.Since(start).Round(time.Millisecond)
	msg := strings.TrimSpace(string(out))
	if err != nil {
		slog.Error("guestfs exec failed", "bin", "guestfish", "duration", duration, "error", err, "output", msg)
		if msg != "" {
			return out, fmt.Errorf("guestfish: %w\n%s", err, msg)
		}
		return out, fmt.Errorf("guestfish: %w", err)
	}
	if errMsg := extractGuestfsError(msg); errMsg != "" {
		slog.Error("guestfs exec failed", "bin", "guestfish", "duration", duration, "error", errMsg, "output", msg)
		return out, fmt.Errorf("guestfish: %s", errMsg)
	}
	slog.Info("guestfs exec ok", "bin", "guestfish", "duration", duration, "outputBytes", len(out), "output", truncateLog(msg, 512))
	return out, nil
}

func truncateLog(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func pathError(op, guestPath string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s: %w", op, guestPath, err)
}

// prefixDash ensures every non-empty line in a guestfish script starts
// with '-' so that command errors don't cause the --listen process to
// exit. Without '-', guestfish treats any failed command as fatal and
// terminates the listener, destroying the QEMU appliance.
func prefixDash(script string) string {
	var out strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "-") {
			out.WriteByte('-')
		}
		out.WriteString(trimmed)
		out.WriteByte('\n')
	}
	return out.String()
}

// extractGuestfsError scans combined guestfish output for error lines
// produced when a '-'-prefixed command fails (the process stays alive
// but prints "libguestfs: error: ..." to stderr). Returns the first
// error message found, or "" if none.
func extractGuestfsError(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "libguestfs: error:") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}
