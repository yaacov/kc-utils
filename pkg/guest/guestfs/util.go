//go:build linux

package guestfs

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// guestfishBinary returns virt-guestfish when present (RHEL/UBI symlink so
// argv[0] satisfies the winsupport NTFS allowlist), else guestfish.
var (
	guestfishBinOnce sync.Once
	guestfishBin     string
)

func guestfishBinary() string {
	guestfishBinOnce.Do(func() {
		if _, err := exec.LookPath("virt-guestfish"); err == nil {
			guestfishBin = "virt-guestfish"
			return
		}
		guestfishBin = "guestfish"
	})
	return guestfishBin
}

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
	return runGuestfsCmdWithStdin(nil, name, args...)
}

func runGuestfsCmdWithStdin(stdin []byte, name string, args ...string) ([]byte, error) {
	slog.Info("guestfs exec", "bin", name, "args", args, "stdinBytes", len(stdin))
	start := time.Now()
	cmd := exec.Command(name, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	duration := time.Since(start).Round(time.Millisecond)
	msg := strings.TrimSpace(string(out))
	if err != nil {
		slog.Error("guestfs exec failed", "bin", name, "duration", duration, "error", err, "output", msg)
		if msg != "" {
			return out, fmt.Errorf("%s: %w\n%s", name, err, msg)
		}
		return out, fmt.Errorf("%s: %w", name, err)
	}
	if errMsg := extractGuestfsError(msg); errMsg != "" {
		slog.Error("guestfs exec failed", "bin", name, "duration", duration, "error", errMsg, "output", msg)
		return out, fmt.Errorf("%s: %s", name, errMsg)
	}
	slog.Info("guestfs exec ok", "bin", name, "duration", duration, "outputBytes", len(out), "output", truncateLog(msg, 512))
	return out, nil
}

func runGuestfishScript(args []string, script string) ([]byte, error) {
	bin := guestfishBinary()
	safe := prefixDash(script)
	slog.Info("guestfs exec",
		"bin", bin,
		"args", args,
		"scriptLines", strings.Count(safe, "\n"),
		"script", truncateLog(strings.TrimSpace(safe), 256),
	)
	start := time.Now()
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(safe)
	out, err := cmd.CombinedOutput()
	duration := time.Since(start).Round(time.Millisecond)
	msg := strings.TrimSpace(string(out))
	if err != nil {
		slog.Error("guestfs exec failed", "bin", bin, "duration", duration, "error", err, "output", msg)
		if msg != "" {
			return out, fmt.Errorf("guestfish: %w\n%s", err, msg)
		}
		return out, fmt.Errorf("guestfish: %w", err)
	}
	if errMsg := extractGuestfsError(msg); errMsg != "" {
		slog.Error("guestfs exec failed", "bin", bin, "duration", duration, "error", errMsg, "output", msg)
		return out, fmt.Errorf("guestfish: %s", errMsg)
	}
	slog.Info("guestfs exec ok", "bin", bin, "duration", duration, "outputBytes", len(out), "output", truncateLog(msg, 512))
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
	errs := extractAllGuestfsErrors(output)
	if len(errs) == 0 {
		return ""
	}
	return errs[0]
}

// extractAllGuestfsErrors returns every libguestfs error line in output.
func extractAllGuestfsErrors(output string) []string {
	var errs []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "libguestfs: error:") {
			errs = append(errs, strings.TrimSpace(line))
		}
	}
	return errs
}

// runGuestfishScriptSoft is like runGuestfishScript but does not fail when
// '-'-prefixed commands print libguestfs errors (used by root probe, where
// missing OS marker paths are expected). Process exit errors still fail.
func runGuestfishScriptSoft(args []string, script string) ([]byte, error) {
	bin := guestfishBinary()
	safe := prefixDash(script)
	slog.Info("guestfs exec",
		"bin", bin,
		"args", args,
		"scriptLines", strings.Count(safe, "\n"),
		"script", truncateLog(strings.TrimSpace(safe), 256),
		"soft", true,
	)
	start := time.Now()
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(safe)
	out, err := cmd.CombinedOutput()
	duration := time.Since(start).Round(time.Millisecond)
	msg := strings.TrimSpace(string(out))
	if err != nil {
		slog.Error("guestfs exec failed", "bin", bin, "duration", duration, "error", err, "output", msg, "soft", true)
		if msg != "" {
			return out, fmt.Errorf("guestfish: %w\n%s", err, msg)
		}
		return out, fmt.Errorf("guestfish: %w", err)
	}
	if errs := extractAllGuestfsErrors(msg); len(errs) > 0 {
		slog.Warn("guestfs soft script reported errors",
			"bin", bin,
			"duration", duration,
			"errors", len(errs),
			"first", errs[0],
			"output", truncateLog(msg, 1024),
		)
	} else {
		slog.Info("guestfs exec ok", "bin", bin, "duration", duration, "outputBytes", len(out), "output", truncateLog(msg, 512), "soft", true)
	}
	return out, nil
}
