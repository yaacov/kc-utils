package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// fallbackBinfmtMagic maps qemu-user-static basenames to a binfmt_misc
// register line (F-flag already set). Used when packaged /usr/lib/binfmt.d
// conf is missing. Magic/mask match qemu's qemu-binfmt-conf.sh.
var fallbackBinfmtMagic = map[string]string{
	"qemu-x86_64-static":  `:qemu-x86_64:M::\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00:\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:%s:CF`,
	"qemu-i386-static":    `:qemu-i386:M::\x7fELF\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x03\x00:\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:%s:CF`,
	"qemu-aarch64-static": `:qemu-aarch64:M::\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\xb7\x00:\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:%s:CF`,
}

// ensureBinfmtFFlag returns a register line with the F (fix-binary) flag set
// so the kernel opens the interpreter before chroot. Empty / comments yield ok=false.
func ensureBinfmtFFlag(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", false
	}
	if !strings.HasPrefix(line, ":") {
		return "", false
	}
	parts := strings.Split(line, ":")
	// "", name, type, offset, magic, mask, interpreter, flags
	if len(parts) < 7 {
		return "", false
	}
	flags := ""
	if len(parts) >= 8 {
		flags = parts[7]
	}
	if !strings.Contains(flags, "F") {
		flags += "F"
	}
	out := strings.Join(parts[:7], ":") + ":" + flags
	return out, true
}

// binfmtInterpreter is the interpreter path from a register line.
func binfmtInterpreter(line string) string {
	parts := strings.Split(line, ":")
	if len(parts) < 7 {
		return ""
	}
	return parts[6]
}

// binfmtName is the :name: field of a register line.
func binfmtName(line string) string {
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// binfmtTargetsNative reports whether registering this interpreter would
// intercept the appliance's own ISA (and break native exec).
func binfmtTargetsNative(interpreter, goarch string) bool {
	base := filepath.Base(interpreter)
	switch goarch {
	case "arm64":
		return strings.Contains(base, "aarch64")
	case "amd64":
		return strings.Contains(base, "x86_64") ||
			strings.Contains(base, "i386") ||
			strings.Contains(base, "i486")
	default:
		return false
	}
}

func fallbackBinfmtLine(interpreter string) (string, bool) {
	tmpl, ok := fallbackBinfmtMagic[filepath.Base(interpreter)]
	if !ok {
		return "", false
	}
	return fmt.Sprintf(tmpl, interpreter), true
}

func parseBinfmtConf(r io.Reader) []string {
	var lines []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line, ok := ensureBinfmtFFlag(sc.Text())
		if ok {
			lines = append(lines, line)
		}
	}
	return lines
}

func readBinfmtConfFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseBinfmtConf(f), nil
}

// collectBinfmtRegisterLines builds unique F-flag register lines for foreign
// qemu-user-static interpreters. Packaged conf wins; fallback magics fill gaps.
func collectBinfmtRegisterLines(confDirs, interpreters []string, goarch string) ([]string, error) {
	want := map[string]string{} // basename -> absolute path
	for _, p := range interpreters {
		want[filepath.Base(p)] = p
	}

	var lines []string
	seenName := map[string]bool{}
	covered := map[string]bool{}

	for _, dir := range confDirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*.conf"))
		if err != nil {
			return nil, err
		}
		for _, path := range matches {
			parsed, err := readBinfmtConfFile(path)
			if err != nil {
				return nil, err
			}
			for _, line := range parsed {
				interp := binfmtInterpreter(line)
				if interp == "" || binfmtTargetsNative(interp, goarch) {
					continue
				}
				base := filepath.Base(interp)
				if _, ok := want[base]; !ok {
					continue
				}
				name := binfmtName(line)
				if name == "" || seenName[name] {
					continue
				}
				seenName[name] = true
				covered[base] = true
				lines = append(lines, line)
			}
		}
	}

	for base, interp := range want {
		if covered[base] || binfmtTargetsNative(interp, goarch) {
			continue
		}
		line, ok := fallbackBinfmtLine(interp)
		if !ok {
			continue
		}
		name := binfmtName(line)
		if name == "" || seenName[name] {
			continue
		}
		seenName[name] = true
		lines = append(lines, line)
	}
	return lines, nil
}
