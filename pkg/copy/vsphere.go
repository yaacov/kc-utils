package copy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	vimtypes "github.com/vmware/govmomi/vim25/types"
)

// DiskURL pairs a download URL with its source VMDK path and size.
type DiskURL struct {
	URL      string
	DiskPath string // [datastore] vm/disk.vmdk
	Size     int64  // bytes
}

// Lease wraps an NFC export lease and its per-disk download URLs.
type Lease struct {
	nfcLease *nfc.Lease
	client   *govmomi.Client
	DiskURLs []DiskURL
	cancel   context.CancelFunc
}

// ExportVM starts an NFC export of the named VM and returns a Lease with
// per-disk HTTPS download URLs. The caller must call Complete or Abort.
func ExportVM(ctx context.Context, libvirtURL, vmName string) (*Lease, error) {
	client, err := vsphereConnect(ctx, libvirtURL)
	if err != nil {
		return nil, err
	}

	finder := find.NewFinder(client.Client, true)
	if dcName := vsphereDatacenter(libvirtURL); dcName != "" {
		if dc, dcErr := finder.Datacenter(ctx, dcName); dcErr == nil {
			finder.SetDatacenter(dc)
		}
	}

	vm, err := finder.VirtualMachine(ctx, vmName)
	if err != nil {
		_ = client.Logout(ctx)
		return nil, fmt.Errorf("find VM %q: %w", vmName, err)
	}

	lease, err := vm.Export(ctx)
	if err != nil {
		_ = client.Logout(ctx)
		return nil, fmt.Errorf("export VM %q: %w", vmName, err)
	}

	info, err := lease.Wait(ctx, nil)
	if err != nil {
		_ = client.Logout(ctx)
		return nil, fmt.Errorf("wait for NFC lease: %w", err)
	}

	var vmProps mo.VirtualMachine
	pc := property.DefaultCollector(client.Client)
	if err := pc.RetrieveOne(ctx, vm.Reference(), []string{"config.hardware.device"}, &vmProps); err != nil {
		_ = lease.Abort(ctx, nil)
		_ = client.Logout(ctx)
		return nil, fmt.Errorf("read VM devices: %w", err)
	}

	diskURLs := mapDiskURLs(info, vmProps.Config)
	slog.Info("NFC export lease acquired",
		"vm", vmName,
		"disks", len(diskURLs),
	)

	updaterCtx, cancel := context.WithCancel(ctx)
	go lease.StartUpdater(updaterCtx, info)

	return &Lease{
		nfcLease: lease,
		client:   client,
		DiskURLs: diskURLs,
		cancel:   cancel,
	}, nil
}

// Complete marks the lease as successfully finished.
func (l *Lease) Complete(ctx context.Context) error {
	l.cancel()
	err := l.nfcLease.Complete(ctx)
	_ = l.client.Logout(ctx)
	return err
}

// Abort cancels the lease.
func (l *Lease) Abort(ctx context.Context) error {
	l.cancel()
	err := l.nfcLease.Abort(ctx, nil)
	_ = l.client.Logout(ctx)
	return err
}

// mapDiskURLs pairs NFC device URLs with VMDK backing file paths.
func mapDiskURLs(info *nfc.LeaseInfo, config *vimtypes.VirtualMachineConfigInfo) []DiskURL {
	devicePaths := map[string]string{}
	if config != nil {
		for _, dev := range config.Hardware.Device {
			disk, ok := dev.(*vimtypes.VirtualDisk)
			if !ok {
				continue
			}
			key := fmt.Sprintf("%d", disk.Key)
			if backing, ok := disk.Backing.(*vimtypes.VirtualDiskFlatVer2BackingInfo); ok {
				devicePaths[key] = backing.FileName
			}
		}
	}

	var urls []DiskURL
	for _, item := range info.Items {
		diskPath := devicePaths[item.DeviceId]
		if diskPath == "" {
			diskPath = item.Path
		}
		urls = append(urls, DiskURL{
			URL:      item.URL.String(),
			DiskPath: diskPath,
			Size:     item.Size,
		})
	}
	return urls
}

// --- vSphere connection helpers (standalone, no pkg/v2v dependency) ---

func vsphereConnect(ctx context.Context, libvirtURL string) (*govmomi.Client, error) {
	sdk, insecure, err := vsphereSdkURL(libvirtURL)
	if err != nil {
		return nil, err
	}
	user, password, err := vsphereCredentials()
	if err != nil {
		return nil, err
	}
	sdk.User = url.UserPassword(user, password)
	client, err := govmomi.NewClient(ctx, sdk, insecure)
	if err != nil {
		return nil, fmt.Errorf("connect to vSphere: %w", err)
	}
	return client, nil
}

func vsphereSdkURL(libvirtURL string) (*url.URL, bool, error) {
	u, err := url.Parse(libvirtURL)
	if err != nil {
		return nil, false, fmt.Errorf("parse libvirt URL: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return nil, false, fmt.Errorf("libvirt URL has no host: %s", libvirtURL)
	}
	if p := u.Port(); p != "" {
		host = net.JoinHostPort(host, p)
	}
	insecure := strings.Contains(u.RawQuery, "no_verify=1") ||
		strings.Contains(u.RawQuery, "no_verify")
	sdk := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/sdk",
	}
	return sdk, insecure, nil
}

func vsphereCredentials() (user, password string, err error) {
	userBytes, err := os.ReadFile("/etc/secret/accessKeyId")
	if err != nil {
		return "", "", fmt.Errorf("read vSphere username: %w", err)
	}
	passBytes, err := os.ReadFile("/etc/secret/secretKey")
	if err != nil {
		return "", "", fmt.Errorf("read vSphere password: %w", err)
	}
	user = strings.TrimSpace(string(userBytes))
	password = strings.TrimSpace(string(passBytes))
	if user == "" || password == "" {
		return "", "", fmt.Errorf("vSphere credentials in /etc/secret are empty")
	}
	return user, password, nil
}

func vsphereDatacenter(libvirtURL string) string {
	u, err := url.Parse(libvirtURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
