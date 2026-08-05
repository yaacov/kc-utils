package mock

import "testing"

func TestMockHiveSetAndGet(t *testing.T) {
	h := NewMockHive()
	h.SetString("Software\\Test", "Name", "hello")
	got, err := h.GetString("Software\\Test", "Name")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestMockHiveSetDWORD(t *testing.T) {
	h := NewMockHive()
	h.SetDWORD("Software\\Test", "Count", 42)
	got, err := h.GetDWORD("Software\\Test", "Count")
	if err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestMockHiveOpsRecording(t *testing.T) {
	h := NewMockHive()
	h.CreateKey("Software\\New")
	h.SetString("Software\\New", "Val", "x")
	h.DeleteValue("Software\\New", "Val")
	if len(h.Ops) != 3 {
		t.Fatalf("got %d ops, want 3", len(h.Ops))
	}
	if h.Ops[0].Action != "create-key" {
		t.Errorf("first op = %q, want create-key", h.Ops[0].Action)
	}
	if h.Ops[2].Action != "delete-value" {
		t.Errorf("third op = %q, want delete-value", h.Ops[2].Action)
	}
}

func TestMockHiveKeyExists(t *testing.T) {
	h := NewMockHive()
	if h.KeyExists("missing") {
		t.Error("non-existent key should not exist")
	}
	h.CreateKey("exists")
	if !h.KeyExists("exists") {
		t.Error("created key should exist")
	}
}

func TestEnumKeys(t *testing.T) {
	h := NewMockHive()
	h.CreateKey(`Software\Microsoft`)
	h.CreateKey(`Software\Microsoft\Windows`)
	h.CreateKey(`Software\Microsoft\Office`)
	h.CreateKey(`Software\Other`)

	keys, err := h.EnumKeys(`Software\Microsoft`)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, k := range keys {
		got[k] = true
	}
	if !got["Windows"] || !got["Office"] {
		t.Errorf("EnumKeys = %v, want [Windows, Office]", keys)
	}
	if got["Other"] {
		t.Error("EnumKeys should not return keys outside the path")
	}
	if len(keys) != 2 {
		t.Errorf("EnumKeys returned %d keys, want 2", len(keys))
	}
}

func TestEnumKeysEmpty(t *testing.T) {
	h := NewMockHive()
	keys, err := h.EnumKeys(`Software`)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Errorf("EnumKeys on empty = %v, want []", keys)
	}
}

func TestMultiStringRoundTrip(t *testing.T) {
	h := NewMockHive()
	vals := []string{"first", "second", "third"}
	h.SetMultiString(`Key`, "Multi", vals)
	got, err := h.GetMultiString(`Key`, "Multi")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(vals) {
		t.Fatalf("len = %d, want %d", len(got), len(vals))
	}
	for i, v := range vals {
		if got[i] != v {
			t.Errorf("got[%d] = %q, want %q", i, got[i], v)
		}
	}
}

func TestExpandStringRoundTrip(t *testing.T) {
	h := NewMockHive()
	h.SetExpandString(`Key`, "Path", `%SystemRoot%\inf`)
	got, err := h.GetString(`Key`, "Path")
	if err != nil {
		t.Fatal(err)
	}
	if got != `%SystemRoot%\inf` {
		t.Errorf("got %q, want %%SystemRoot%%\\inf", got)
	}
	_, typ, err := h.GetValue(`Key`, "Path")
	if err != nil {
		t.Fatal(err)
	}
	if typ != 2 {
		t.Errorf("type = %d, want REG_EXPAND_SZ (2)", typ)
	}
}

func TestBinaryRoundTrip(t *testing.T) {
	h := NewMockHive()
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	h.SetBinary(`Key`, "Bin", data)
	got, typ, err := h.GetValue(`Key`, "Bin")
	if err != nil {
		t.Fatal(err)
	}
	if typ != 3 {
		t.Errorf("type = %d, want REG_BINARY (3)", typ)
	}
	if len(got) != 4 || got[0] != 0xDE || got[3] != 0xEF {
		t.Errorf("data = %v, want [DE AD BE EF]", got)
	}
}

func TestEnumValues(t *testing.T) {
	h := NewMockHive()
	h.SetString(`Key`, "A", "val-a")
	h.SetDWORD(`Key`, "B", 99)
	vals, err := h.EnumValues(`Key`)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 2 {
		t.Fatalf("EnumValues returned %d, want 2", len(vals))
	}
}

func TestEnumValuesEmpty(t *testing.T) {
	h := NewMockHive()
	vals, err := h.EnumValues(`Missing`)
	if err != nil {
		t.Fatal(err)
	}
	if vals != nil {
		t.Errorf("EnumValues on missing key = %v, want nil", vals)
	}
}

func TestDeleteKey(t *testing.T) {
	h := NewMockHive()
	h.CreateKey(`Software\Test`)
	h.SetString(`Software\Test`, "Val", "x")
	h.DeleteKey(`Software\Test`)
	if h.KeyExists(`Software\Test`) {
		t.Error("key should not exist after DeleteKey")
	}
	_, err := h.GetString(`Software\Test`, "Val")
	if err == nil {
		t.Error("values should be gone after DeleteKey")
	}
}

func TestGetNotFound(t *testing.T) {
	h := NewMockHive()
	if _, err := h.GetString("X", "Y"); err == nil {
		t.Error("GetString on missing should error")
	}
	if _, err := h.GetDWORD("X", "Y"); err == nil {
		t.Error("GetDWORD on missing should error")
	}
	if _, err := h.GetMultiString("X", "Y"); err == nil {
		t.Error("GetMultiString on missing should error")
	}
	if _, _, err := h.GetValue("X", "Y"); err == nil {
		t.Error("GetValue on missing should error")
	}
}
