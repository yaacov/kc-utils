package copy

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

type stubNFCDownloader struct {
	urls []string
	body []byte
}

func (s *stubNFCDownloader) downloadURL(_ context.Context, rawURL string) (io.ReadCloser, error) {
	s.urls = append(s.urls, rawURL)
	return io.NopCloser(bytes.NewReader(s.body)), nil
}

func TestCopyDiskFromDownloaderUsesLeaseURL(t *testing.T) {
	vmdk := buildTestVMDK(t, 2, 0, make([]byte, 2*sectorSize))
	dl := &stubNFCDownloader{body: vmdk}

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "disk.img")
	target := Target{Index: 0, Path: targetPath, IsBlockDev: false}

	const nfcURL = "https://10.6.46.29/nfc/lease/disk-0.vmdk"
	disk := DiskURL{URL: nfcURL, DiskPath: "[ds] vm/disk.vmdk", Size: int64(len(vmdk))}

	if err := copyDiskFromDownloader(context.Background(), dl, disk, target, nil); err != nil {
		t.Fatalf("copyDiskFromDownloader: %v", err)
	}
	if len(dl.urls) != 1 || dl.urls[0] != nfcURL {
		t.Fatalf("urls = %v, want [%q]", dl.urls, nfcURL)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty output file")
	}
}

func TestLeaseDownloadURLInvalidURL(t *testing.T) {
	u, err := soap.ParseURL("https://vcenter.example.com/sdk")
	if err != nil {
		t.Fatal(err)
	}
	lease := &Lease{
		client: &govmomi.Client{
			Client: &vim25.Client{
				Client: soap.NewClient(u, true),
			},
		},
	}
	_, err = lease.downloadURL(context.Background(), "://bad")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}
