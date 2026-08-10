package driversource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseINFRequirements(t *testing.T) {
	dir := t.TempDir()
	inf := filepath.Join(dir, "netkvm.inf")
	writeFile(t, inf, `[Version]
Signature="$WINDOWS NT$"
CatalogFile = netkvm.cat

[SourceDisksFiles]
netkvm.sys  = 1,,
netkvmp.exe = 1,,

[Other]
IgnoreMe.sys = 1,,
`)

	catalogs, companions, err := parseINFRequirements(inf, "x86_64")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogs) != 1 || catalogs[0] != "netkvm.cat" {
		t.Fatalf("catalogs=%v", catalogs)
	}
	want := []string{"netkvm.sys", "netkvmp.exe"}
	if len(companions) != len(want) {
		t.Fatalf("companions=%v want %v", companions, want)
	}
	for i, name := range want {
		if companions[i] != name {
			t.Fatalf("companions[%d]=%q want %q", i, companions[i], name)
		}
	}
}

func TestParseINFRequirementsDecorated(t *testing.T) {
	dir := t.TempDir()
	inf := filepath.Join(dir, "driver.inf")
	writeFile(t, inf, `[Version]
CatalogFile = common.cat
CatalogFile.ntamd64 = amd64.cat
CatalogFile.ntx86 = x86.cat

[SourceDisksFiles]
shared.sys = 1,,

[SourceDisksFiles.amd64]
amd64only.sys = 1,amd64

[SourceDisksFiles.x86]
x86only.sys = 1,x86
`)

	catalogs, companions, err := parseINFRequirements(inf, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	catSet := map[string]bool{}
	for _, c := range catalogs {
		catSet[c] = true
	}
	if !catSet["common.cat"] || !catSet["amd64.cat"] {
		t.Fatalf("catalogs=%v", catalogs)
	}
	if catSet["x86.cat"] {
		t.Fatalf("x86 catalog should not apply to amd64: %v", catalogs)
	}
	compSet := map[string]bool{}
	for _, c := range companions {
		compSet[filepath.ToSlash(c)] = true
	}
	if !compSet["shared.sys"] || !compSet["amd64/amd64only.sys"] {
		t.Fatalf("companions=%v", companions)
	}
	if compSet["x86/x86only.sys"] || compSet["x86only.sys"] {
		t.Fatalf("x86 companions should not apply: %v", companions)
	}
}

func TestParseSourceDisksEntrySubdirAndEscape(t *testing.T) {
	rel := parseSourceDisksEntry(`foo.sys = 1,subdir\bits,`)
	if filepath.ToSlash(rel) != "subdir/bits/foo.sys" {
		t.Fatalf("rel=%q", rel)
	}
	if parseSourceDisksEntry(`evil.sys = 1,..\..\etc,`) != "" {
		t.Fatal("expected reject path escape")
	}
	if filepath.Separator == '/' && parseSourceDisksEntry(`evil.sys = 1,/etc,`) != "" {
		t.Fatal("expected reject absolute subdir")
	}
}

func TestFilterCompleteKeepsCompletePackage(t *testing.T) {
	dir := t.TempDir()
	inf := filepath.Join(dir, "viorng.inf")
	writeFile(t, inf, `[Version]
CatalogFile=viorng.cat
[SourceDisksFiles]
viorng.sys = 1,,
viorngum.dll = 1
`)
	writeFile(t, filepath.Join(dir, "viorng.cat"), "cat")
	writeFile(t, filepath.Join(dir, "viorng.sys"), "sys")
	writeFile(t, filepath.Join(dir, "viorngum.dll"), "dll")
	writeFile(t, filepath.Join(dir, "noise.exe"), "noise")

	kept := FilterComplete([]DriverFile{{
		Name:    "viorng",
		SrcPath: dir,
		InfPath: inf,
		Arch:    "x86_64",
	}})
	if len(kept) != 1 {
		t.Fatalf("kept=%d", len(kept))
	}
	basenames := map[string]bool{}
	for _, f := range kept[0].Files {
		basenames[filepath.Base(f)] = true
	}
	for _, name := range []string{"viorng.inf", "viorng.cat", "viorng.sys", "viorngum.dll"} {
		if !basenames[name] {
			t.Fatalf("missing staged file %s in %#v", name, kept[0].Files)
		}
	}
	if basenames["noise.exe"] {
		t.Fatal("unexpected sibling copied into Files")
	}
}

func TestFilterCompleteDecoratedWithSubdir(t *testing.T) {
	dir := t.TempDir()
	inf := filepath.Join(dir, "netkvm.inf")
	writeFile(t, inf, `[Version]
CatalogFile.ntamd64 = netkvm.cat

[SourceDisksFiles.amd64]
netkvm.sys = 1,bin
netkvmp.exe = 1,bin
`)
	writeFile(t, filepath.Join(dir, "netkvm.cat"), "cat")
	writeFile(t, filepath.Join(dir, "bin", "netkvm.sys"), "sys")
	writeFile(t, filepath.Join(dir, "bin", "netkvmp.exe"), "exe")

	kept := FilterComplete([]DriverFile{{
		Name:    "netkvm",
		SrcPath: dir,
		InfPath: inf,
		Arch:    "x86_64",
	}})
	if len(kept) != 1 {
		t.Fatalf("kept=%d", len(kept))
	}
	found := map[string]bool{}
	for _, f := range kept[0].Files {
		rel, err := filepath.Rel(dir, f)
		if err != nil {
			t.Fatal(err)
		}
		found[filepath.ToSlash(rel)] = true
	}
	for _, want := range []string{"netkvm.inf", "netkvm.cat", "bin/netkvm.sys", "bin/netkvmp.exe"} {
		if !found[want] {
			t.Fatalf("missing %s in %#v", want, found)
		}
	}
}

func TestFilterCompleteDropsMissingCompanion(t *testing.T) {
	dir := t.TempDir()
	inf := filepath.Join(dir, "netkvm.inf")
	writeFile(t, inf, `[Version]
CatalogFile=netkvm.cat
[SourceDisksFiles]
netkvm.sys = 1,,
netkvmp.exe = 1,,
`)
	writeFile(t, filepath.Join(dir, "netkvm.cat"), "cat")
	writeFile(t, filepath.Join(dir, "netkvm.sys"), "sys")
	// netkvmp.exe intentionally missing

	kept := FilterComplete([]DriverFile{{
		Name:    "netkvm",
		SrcPath: dir,
		InfPath: inf,
	}})
	if len(kept) != 0 {
		t.Fatalf("expected drop, kept %#v", kept)
	}
}

func TestFilterCompleteKeepsMSI(t *testing.T) {
	dir := t.TempDir()
	msi := filepath.Join(dir, "qemu-ga-x86_64.msi")
	writeFile(t, msi, "msi")
	writeFile(t, filepath.Join(dir, "qemu-ga-i386.msi"), "other")

	kept := FilterComplete([]DriverFile{{
		Name:    "qemu-ga",
		SrcPath: dir,
		InfPath: msi,
	}})
	if len(kept) != 1 {
		t.Fatalf("kept=%d", len(kept))
	}
	if len(kept[0].Files) != 1 || kept[0].Files[0] != msi {
		t.Fatalf("Files=%v", kept[0].Files)
	}
}

func TestFilterCompleteDropsMissingMSI(t *testing.T) {
	kept := FilterComplete([]DriverFile{{
		Name:    "qemu-ga",
		InfPath: filepath.Join(t.TempDir(), "missing.msi"),
	}})
	if len(kept) != 0 {
		t.Fatalf("expected drop, kept %#v", kept)
	}
}

func TestFilterCompleteINFOnlyNullDriver(t *testing.T) {
	dir := t.TempDir()
	inf := filepath.Join(dir, "qemufwcfg.inf")
	writeFile(t, inf, `[Version]
Class=System
CatalogFile = qemufwcfg.cat
[SourceDisksFiles]
`)
	writeFile(t, filepath.Join(dir, "qemufwcfg.cat"), "cat")

	kept := FilterComplete([]DriverFile{{
		Name:    "qemufwcfg",
		SrcPath: dir,
		InfPath: inf,
	}})
	if len(kept) != 1 {
		t.Fatalf("kept=%d", len(kept))
	}
	joined := strings.Join(kept[0].Files, ",")
	if !strings.Contains(joined, "qemufwcfg.inf") || !strings.Contains(joined, "qemufwcfg.cat") {
		t.Fatalf("Files=%v", kept[0].Files)
	}
}
