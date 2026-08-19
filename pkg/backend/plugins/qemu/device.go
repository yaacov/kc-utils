//go:build unix

package qemu

import "fmt"

// DeviceRead reads size bytes at offset from a raw appliance device (e.g.
// /dev/vda1). Device paths are appliance-absolute and not rebased.
func (b *Backend) DeviceRead(device string, offset int64, size int) ([]byte, error) {
	data, err := b.session.client.pread(device, offset, size)
	if err != nil {
		return nil, err
	}
	if len(data) != size {
		return nil, fmt.Errorf("device read %s: got %d bytes, want %d", device, len(data), size)
	}
	return data, nil
}

// DeviceWrite writes data at offset to a raw appliance device.
func (b *Backend) DeviceWrite(device string, offset int64, data []byte) error {
	n, err := b.session.client.pwrite(device, offset, data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("device write %s: wrote %d bytes, want %d", device, n, len(data))
	}
	return nil
}
