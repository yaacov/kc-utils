//go:build unix

package luks

import (
	"fmt"
	"os/exec"
)

// Open decrypts a LUKS partition using the given key file
// and maps it to the given name under /dev/mapper/.
func Open(device, keyFile, mapperName string) (string, error) {
	args := []string{"open", "--type", "luks", device, mapperName}
	if keyFile != "" {
		args = append(args, "--key-file", keyFile)
	}

	if err := exec.Command("cryptsetup", args...).Run(); err != nil {
		return "", fmt.Errorf("cryptsetup open %s: %w", device, err)
	}
	return "/dev/mapper/" + mapperName, nil
}

// Close removes the device mapper entry.
func Close(mapperName string) error {
	return exec.Command("cryptsetup", "close", mapperName).Run()
}
