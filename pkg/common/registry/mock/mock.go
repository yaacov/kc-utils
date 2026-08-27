package mock

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
)

// Op records a write operation.
type Op struct {
	Action string
	Path   string
	Name   string
	Value  interface{}
}

// MockHive is an in-memory registry hive for testing.
type MockHive struct {
	Keys   map[string]bool
	Values map[string]map[string]registry.ValueEntry
	Ops    []Op
}

// NewMockHive returns an empty in-memory hive.
func NewMockHive() *MockHive {
	return &MockHive{
		Keys:   make(map[string]bool),
		Values: make(map[string]map[string]registry.ValueEntry),
	}
}

func (h *MockHive) ensureKey(path string) {
	h.Keys[path] = true
	if h.Values[path] == nil {
		h.Values[path] = make(map[string]registry.ValueEntry)
	}
}

func (h *MockHive) KeyExists(path string) bool { return h.Keys[path] }

func (h *MockHive) EnumKeys(path string) ([]string, error) {
	var result []string
	prefix := path + "\\"
	for k := range h.Keys {
		if strings.HasPrefix(k, prefix) {
			rest := k[len(prefix):]
			if !strings.Contains(rest, "\\") {
				result = append(result, rest)
			}
		}
	}
	return result, nil
}

func (h *MockHive) GetString(path, name string) (string, error) {
	if vals, ok := h.Values[path]; ok {
		if v, ok := vals[name]; ok {
			return string(v.Data), nil
		}
	}
	return "", fmt.Errorf("not found: %s\\%s", path, name)
}

func (h *MockHive) GetDWORD(path, name string) (uint32, error) {
	if vals, ok := h.Values[path]; ok {
		if v, ok := vals[name]; ok && len(v.Data) >= 4 {
			return binary.LittleEndian.Uint32(v.Data[:4]), nil
		}
	}
	return 0, fmt.Errorf("not found: %s\\%s", path, name)
}

func (h *MockHive) GetMultiString(path, name string) ([]string, error) {
	if vals, ok := h.Values[path]; ok {
		if v, ok := vals[name]; ok {
			return strings.Split(string(v.Data), "\x00"), nil
		}
	}
	return nil, fmt.Errorf("not found: %s\\%s", path, name)
}

func (h *MockHive) GetValue(path, name string) ([]byte, int, error) {
	if vals, ok := h.Values[path]; ok {
		if v, ok := vals[name]; ok {
			return v.Data, v.Type, nil
		}
	}
	return nil, 0, fmt.Errorf("not found: %s\\%s", path, name)
}

func (h *MockHive) EnumValues(path string) ([]registry.ValueEntry, error) {
	vals, ok := h.Values[path]
	if !ok {
		return nil, nil
	}
	var result []registry.ValueEntry
	for _, v := range vals {
		result = append(result, v)
	}
	return result, nil
}

func (h *MockHive) CreateKey(path string) {
	h.ensureKey(path)
	h.Ops = append(h.Ops, Op{"create-key", path, "", nil})
}

func (h *MockHive) DeleteKey(path string) {
	delete(h.Keys, path)
	delete(h.Values, path)
	h.Ops = append(h.Ops, Op{"delete-key", path, "", nil})
}

func (h *MockHive) SetString(path, name, value string) {
	h.ensureKey(path)
	h.Values[path][name] = registry.ValueEntry{Name: name, Type: registry.REG_SZ, Data: []byte(value)}
	h.Ops = append(h.Ops, Op{"set-string", path, name, value})
}

func (h *MockHive) SetExpandString(path, name, value string) {
	h.ensureKey(path)
	h.Values[path][name] = registry.ValueEntry{Name: name, Type: registry.REG_EXPAND_SZ, Data: []byte(value)}
	h.Ops = append(h.Ops, Op{"set-expand-string", path, name, value})
}

func (h *MockHive) SetDWORD(path, name string, value uint32) {
	h.ensureKey(path)
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)
	h.Values[path][name] = registry.ValueEntry{Name: name, Type: registry.REG_DWORD, Data: data}
	h.Ops = append(h.Ops, Op{"set-dword", path, name, value})
}

func (h *MockHive) SetMultiString(path, name string, values []string) {
	h.ensureKey(path)
	h.Values[path][name] = registry.ValueEntry{Name: name, Type: registry.REG_MULTI_SZ, Data: []byte(strings.Join(values, "\x00"))}
	h.Ops = append(h.Ops, Op{"set-multi-string", path, name, values})
}

func (h *MockHive) SetBinary(path, name string, data []byte) {
	h.ensureKey(path)
	h.Values[path][name] = registry.ValueEntry{Name: name, Type: registry.REG_BINARY, Data: data}
	h.Ops = append(h.Ops, Op{"set-binary", path, name, data})
}

func (h *MockHive) DeleteValue(path, name string) {
	if vals, ok := h.Values[path]; ok {
		delete(vals, name)
	}
	h.Ops = append(h.Ops, Op{"delete-value", path, name, nil})
}

func (h *MockHive) Save() error  { return nil }
func (h *MockHive) Close() error { return nil }
