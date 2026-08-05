package registry

import "github.com/yaacov/kc-utils/pkg/common/plugin"

// Editor reads and writes Windows registry hive files.
type Editor interface {
	OpenHive(hivePath string) (Hive, error)
}

// Hive represents an opened registry hive with read/write access.
type Hive interface {
	KeyExists(path string) bool
	EnumKeys(path string) ([]string, error)
	GetString(path, name string) (string, error)
	GetDWORD(path, name string) (uint32, error)
	GetMultiString(path, name string) ([]string, error)
	GetValue(path, name string) ([]byte, int, error)
	EnumValues(path string) ([]ValueEntry, error)

	CreateKey(path string)
	DeleteKey(path string)
	SetString(path, name, value string)
	SetExpandString(path, name, value string)
	SetDWORD(path, name string, value uint32)
	SetMultiString(path, name string, values []string)
	SetBinary(path, name string, data []byte)
	DeleteValue(path, name string)

	Save() error
	Close() error
}

// ValueEntry represents a registry value.
type ValueEntry struct {
	Name string
	Type int
	Data []byte
}

const (
	REG_SZ        = 1
	REG_EXPAND_SZ = 2
	REG_BINARY    = 3
	REG_DWORD     = 4
	REG_MULTI_SZ  = 7
)

// Editors is the global registry of Editor implementations.
var Editors = plugin.NewRegistry[string, Editor]()
