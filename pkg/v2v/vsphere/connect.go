package vsphere

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/vmware/govmomi"
)

func sdkURL(libvirtURL string) (*url.URL, bool, error) {
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

func credentials(libvirtURL string) (user, password string, err error) {
	u, err := url.Parse(libvirtURL)
	if err != nil {
		return "", "", err
	}
	user = u.User.Username()
	if userBytes, readErr := os.ReadFile("/etc/secret/accessKeyId"); readErr == nil {
		if fromSecret := strings.TrimSpace(string(userBytes)); fromSecret != "" {
			user = fromSecret
		}
	}
	passBytes, err := os.ReadFile("/etc/secret/secretKey")
	if err != nil {
		return "", "", fmt.Errorf("read vSphere password: %w", err)
	}
	password = strings.TrimSpace(string(passBytes))
	if user == "" || password == "" {
		return "", "", fmt.Errorf("vSphere credentials are empty")
	}
	return user, password, nil
}

func connect(ctx context.Context, libvirtURL string) (*govmomi.Client, error) {
	sdk, insecure, err := sdkURL(libvirtURL)
	if err != nil {
		return nil, err
	}
	user, password, err := credentials(libvirtURL)
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

func datacenterName(libvirtURL string) string {
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
