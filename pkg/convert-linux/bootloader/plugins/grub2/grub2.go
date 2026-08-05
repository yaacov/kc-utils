//go:build linux

package grub2

import (
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/configedit/grub"
	"github.com/yaacov/kc-utils/pkg/convert-linux/bootloader"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Grub2Handler struct{}

func init() {
	bootloader.Handlers.Register("grub2", &Grub2Handler{})
}

func (g *Grub2Handler) Detect(guestRoot string) bool {
	paths := []string{
		filepath.Join(guestRoot, "etc", "default", "grub"),
		filepath.Join(guestRoot, "boot", "grub2", "grub.cfg"),
		filepath.Join(guestRoot, "boot", "grub", "grub.cfg"),
	}
	for _, p := range paths {
		if guest.FileExists(p) {
			return true
		}
	}
	return false
}

func (g *Grub2Handler) GetDefaultKernel(guestRoot string) (string, error) {
	content, err := guest.FileRead(filepath.Join(guestRoot, "etc", "default", "grub"))
	if err != nil {
		return "", err
	}
	cfg := grub.Parse(string(content))
	return cfg.Get("GRUB_DEFAULT"), nil
}

func (g *Grub2Handler) SetDefaultKernel(guestRoot, kernelVersion string) error {
	path := filepath.Join(guestRoot, "etc", "default", "grub")
	content, err := guest.FileRead(path)
	if err != nil {
		return err
	}
	cfg := grub.Parse(string(content))
	// Prefer saved/default by title containing the version; also set DEFAULTKERNEL for RHEL.
	cfg.Set("GRUB_DEFAULT", "0")
	if kernelVersion != "" {
		cfg.Set("DEFAULTKERNEL", "kernel-"+kernelVersion)
	}
	if err := guest.FileWrite(path, []byte(cfg.String()), 0o644); err != nil {
		return err
	}
	return g.RegenerateConfig(guestRoot)
}

func (g *Grub2Handler) AddKernelArg(guestRoot, arg string) error {
	path := filepath.Join(guestRoot, "etc", "default", "grub")
	content, err := guest.FileRead(path)
	if err != nil {
		return err
	}
	cfg := grub.Parse(string(content))
	cfg.AddKernelArg(arg)
	if err := guest.FileWrite(path, []byte(cfg.String()), 0o644); err != nil {
		return err
	}
	return g.RegenerateConfig(guestRoot)
}

func (g *Grub2Handler) RemoveKernelArg(guestRoot, prefix string) error {
	path := filepath.Join(guestRoot, "etc", "default", "grub")
	content, err := guest.FileRead(path)
	if err != nil {
		return err
	}
	cfg := grub.Parse(string(content))
	cfg.RemoveKernelArg(prefix)
	if err := guest.FileWrite(path, []byte(cfg.String()), 0o644); err != nil {
		return err
	}
	return g.RegenerateConfig(guestRoot)
}

func (g *Grub2Handler) RegenerateConfig(guestRoot string) error {
	outCandidates := []string{
		"/boot/grub2/grub.cfg",
		"/boot/grub/grub.cfg",
		"/boot/efi/EFI/redhat/grub.cfg",
		"/boot/efi/EFI/centos/grub.cfg",
		"/boot/efi/EFI/fedora/grub.cfg",
	}
	var outPath string
	for _, rel := range outCandidates {
		if guest.FileExists(filepath.Join(guestRoot, rel)) {
			outPath = rel
			break
		}
	}
	if outPath == "" {
		outPath = "/boot/grub2/grub.cfg"
	}

	for _, bin := range []string{"grub2-mkconfig", "grub-mkconfig"} {
		out, err := guest.RunInGuest(guestRoot, []string{bin, "-o", outPath})
		if err != nil {
			slog.Debug("grub mkconfig via guest runner failed", "bin", bin, "error", err, "output", string(out))
			continue
		}
		slog.Info("regenerated grub config", "bin", bin, "output", outPath)
		return nil
	}
	slog.Debug("grub mkconfig unavailable; /etc/default/grub updated only")
	return nil
}
