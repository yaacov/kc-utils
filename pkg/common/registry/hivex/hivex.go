package hivex

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"www.velocidex.com/golang/regparser"
)

// HivexEditor uses regparser for reads and hivexregedit CLI for writes.
type HivexEditor struct{}

func init() {
	registry.Editors.Register("hivex", &HivexEditor{})
}

func (e *HivexEditor) OpenHive(hivePath string) (registry.Hive, error) {
	f, err := os.Open(hivePath)
	if err != nil {
		return nil, fmt.Errorf("open hive %s: %w", hivePath, err)
	}
	reg, err := regparser.NewRegistry(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("parse hive %s: %w", hivePath, err)
	}
	return &HivexHive{
		hivePath: hivePath,
		file:     f,
		reg:      reg,
		pending:  &bytes.Buffer{},
	}, nil
}

// HivexHive implements registry.Hive using regparser for reads and
// hivexregedit for writes.
type HivexHive struct {
	hivePath string
	file     *os.File
	reg      *regparser.Registry
	pending  *bytes.Buffer
}

func (h *HivexHive) KeyExists(path string) bool {
	return h.reg.OpenKey(path) != nil
}

func (h *HivexHive) EnumKeys(path string) ([]string, error) {
	key := h.reg.OpenKey(path)
	if key == nil {
		return nil, fmt.Errorf("key not found: %s", path)
	}
	var names []string
	for _, sub := range key.Subkeys() {
		names = append(names, sub.Name())
	}
	return names, nil
}

func (h *HivexHive) findValue(path, name string) (*regparser.ValueData, error) {
	key := h.reg.OpenKey(path)
	if key == nil {
		return nil, fmt.Errorf("key not found: %s", path)
	}
	for _, v := range key.Values() {
		if strings.EqualFold(v.ValueName(), name) {
			return v.ValueData(), nil
		}
	}
	return nil, fmt.Errorf("value not found: %s\\%s", path, name)
}

func (h *HivexHive) GetString(path, name string) (string, error) {
	vd, err := h.findValue(path, name)
	if err != nil {
		return "", err
	}
	return vd.String, nil
}

func (h *HivexHive) GetDWORD(path, name string) (uint32, error) {
	vd, err := h.findValue(path, name)
	if err != nil {
		return 0, err
	}
	if len(vd.Data) < 4 {
		return 0, fmt.Errorf("data too short for DWORD")
	}
	return binary.LittleEndian.Uint32(vd.Data[:4]), nil
}

func (h *HivexHive) GetMultiString(path, name string) ([]string, error) {
	vd, err := h.findValue(path, name)
	if err != nil {
		return nil, err
	}
	return strings.Split(vd.String, "\x00"), nil
}

func (h *HivexHive) GetValue(path, name string) ([]byte, int, error) {
	key := h.reg.OpenKey(path)
	if key == nil {
		return nil, 0, fmt.Errorf("key not found: %s", path)
	}
	for _, v := range key.Values() {
		if strings.EqualFold(v.ValueName(), name) {
			return v.ValueData().Data, int(v.Type()), nil
		}
	}
	return nil, 0, fmt.Errorf("value not found: %s\\%s", path, name)
}

func (h *HivexHive) EnumValues(path string) ([]registry.ValueEntry, error) {
	key := h.reg.OpenKey(path)
	if key == nil {
		return nil, fmt.Errorf("key not found: %s", path)
	}
	var entries []registry.ValueEntry
	for _, v := range key.Values() {
		entries = append(entries, registry.ValueEntry{
			Name: v.ValueName(),
			Type: int(v.Type()),
			Data: v.ValueData().Data,
		})
	}
	return entries, nil
}

func (h *HivexHive) initPending() {
	if h.pending.Len() == 0 {
		h.pending.WriteString("Windows Registry Editor Version 5.00\n\n")
	}
}

func (h *HivexHive) writeKeyHeader(path string) {
	h.initPending()
	fmt.Fprintf(h.pending, "[%s]\n", path)
}

func (h *HivexHive) CreateKey(path string) {
	parts := strings.Split(path, `\`)
	for i := range parts {
		ancestor := strings.Join(parts[:i+1], `\`)
		h.writeKeyHeader(ancestor)
		h.pending.WriteString("\n")
	}
}

func (h *HivexHive) DeleteKey(path string) {
	h.initPending()
	fmt.Fprintf(h.pending, "[-%s]\n\n", path)
}

func (h *HivexHive) SetString(path, name, value string) {
	h.writeKeyHeader(path)
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	fmt.Fprintf(h.pending, "\"%s\"=\"%s\"\n\n", name, escaped)
}

func (h *HivexHive) SetExpandString(path, name, value string) {
	h.writeKeyHeader(path)
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	fmt.Fprintf(h.pending, "\"%s\"=str(2):\"%s\"\n\n", name, escaped)
}

func (h *HivexHive) SetDWORD(path, name string, value uint32) {
	h.writeKeyHeader(path)
	fmt.Fprintf(h.pending, "\"%s\"=dword:%08x\n\n", name, value)
}

func (h *HivexHive) SetMultiString(path, name string, values []string) {
	h.writeKeyHeader(path)
	joined := strings.Join(values, "\\0")
	fmt.Fprintf(h.pending, "\"%s\"=str(7):\"%s\\0\"\n\n", name, joined)
}

func (h *HivexHive) SetBinary(path, name string, data []byte) {
	h.writeKeyHeader(path)
	var hexParts []string
	for _, b := range data {
		hexParts = append(hexParts, fmt.Sprintf("%02x", b))
	}
	fmt.Fprintf(h.pending, "\"%s\"=hex:%s\n\n", name, strings.Join(hexParts, ","))
}

func (h *HivexHive) DeleteValue(path, name string) {
	h.writeKeyHeader(path)
	fmt.Fprintf(h.pending, "\"%s\"=-\n\n", name)
}

func (h *HivexHive) Save() error {
	if h.pending.Len() == 0 {
		return nil
	}
	tmpFile, err := os.CreateTemp("", "kc-reg-*.reg")
	if err != nil {
		return fmt.Errorf("creating temp .reg file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(h.pending.Bytes()); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	h.file.Close()

	cmd := exec.Command("hivexregedit", "--merge", h.hivePath, tmpFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("hivexregedit --merge failed: %w\n%s", err, output)
	}

	h.file, err = os.Open(h.hivePath)
	if err != nil {
		return err
	}
	h.reg, err = regparser.NewRegistry(h.file)
	if err != nil {
		return err
	}

	h.pending.Reset()
	return nil
}

func (h *HivexHive) Close() error {
	h.pending.Reset()
	return h.file.Close()
}
