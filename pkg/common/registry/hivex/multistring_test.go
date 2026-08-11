package hivex

import (
	"testing"

	"www.velocidex.com/golang/regparser"
)

func TestMultiStringFromValueDataPrefersMultiSz(t *testing.T) {
	vd := &regparser.ValueData{
		Type:    regparser.REG_MULTI_SZ,
		MultiSz: []string{"PartMgr", "XENFILT"},
		String:  "", // regparser leaves String empty for REG_MULTI_SZ
	}
	got := multiStringFromValueData(vd)
	if len(got) != 2 || got[0] != "PartMgr" || got[1] != "XENFILT" {
		t.Fatalf("got %v, want [PartMgr XENFILT]", got)
	}
}

func TestMultiStringFromValueDataEmptyStringNotFakeEntry(t *testing.T) {
	// Old bug: strings.Split("", "\x00") == [""], which made RemoveFilter a no-op.
	vd := &regparser.ValueData{Type: regparser.REG_MULTI_SZ, String: ""}
	got := multiStringFromValueData(vd)
	if len(got) != 0 {
		t.Fatalf("got %v, want empty slice", got)
	}
}

func TestMultiStringFromValueDataStringFallback(t *testing.T) {
	vd := &regparser.ValueData{Type: regparser.REG_MULTI_SZ, String: "a\x00b\x00"}
	got := multiStringFromValueData(vd)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v, want [a b]", got)
	}
}
