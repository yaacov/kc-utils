package xml

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

type inspectionV2V struct {
	XMLName xml.Name     `xml:"v2v"`
	OS      inspectionOS `xml:"operatingsystem"`
}

type inspectionOS struct {
	Name   string `xml:"name"`
	Distro string `xml:"distro"`
	Osinfo string `xml:"osinfo"`
	Arch   string `xml:"arch"`
}

// WriteInspectionXML writes a minimal virt-v2v inspection XML file from TargetMeta.
func WriteInspectionXML(meta *types.TargetMeta, path string) error {
	osinfo := meta.Inspect.OsinfoID
	if osinfo == "" {
		osinfo = inferOsinfo(&meta.Inspect)
	}
	doc := inspectionV2V{
		OS: inspectionOS{
			Name:   meta.Inspect.ProductName,
			Distro: meta.Inspect.Distro,
			Osinfo: osinfo,
			Arch:   meta.Inspect.Arch,
		},
	}
	if doc.OS.Name == "" {
		doc.OS.Name = meta.Inspect.Distro
	}
	if doc.OS.Arch == "" {
		slog.Warn("guest architecture not set in metadata, defaulting to x86_64")
		doc.OS.Arch = "x86_64"
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	content := xml.Header + string(out) + "\n"
	if err := os.MkdirAll(filepathDir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func filepathDir(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return "."
}

func inferOsinfo(inspect *types.InspectData) string {
	if inspect.Type == "windows" {
		return inferWindowsOsinfo(inspect)
	}
	d := strings.ToLower(inspect.Distro)
	major := inspect.MajorVersion
	switch {
	case strings.Contains(d, "rhel"):
		return fmt.Sprintf("rhel%d", major)
	case strings.Contains(d, "centos"):
		return fmt.Sprintf("centos%d", major)
	case strings.Contains(d, "fedora"):
		return "fedora"
	case strings.Contains(d, "ubuntu"):
		return "ubuntu"
	case strings.Contains(d, "debian"):
		return fmt.Sprintf("debian%d", major)
	case strings.Contains(d, "sles"), strings.Contains(d, "suse"):
		return fmt.Sprintf("sles%d", major)
	default:
		if d != "" {
			return d
		}
		return inspect.Type
	}
}

func inferWindowsOsinfo(inspect *types.InspectData) string {
	name := strings.ToLower(inspect.ProductName)
	switch {
	case strings.Contains(name, "2025"):
		return "win2k25"
	case strings.Contains(name, "2022"):
		return "win2k22"
	case strings.Contains(name, "2019"):
		return "win2k19"
	case strings.Contains(name, "2016"):
		return "win2k16"
	case strings.Contains(name, "11"):
		return "win11"
	case strings.Contains(name, "10"):
		return "win10"
	default:
		return "win10"
	}
}
