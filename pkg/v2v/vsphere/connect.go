package vsphere

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	v2vtls "github.com/yaacov/kc-utils/pkg/v2v/tls"
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
	insecure := v2vtls.InsecureFromQuery(u.RawQuery)
	sdk := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/sdk",
	}
	return sdk, insecure, nil
}

const (
	secretAccessKeyPath = "/etc/secret/accessKeyId"
	secretSecretKeyPath = "/etc/secret/secretKey"
)

// credentials resolves vSphere user/password: explicit values first, then
// Forklift secret files, then the username in libvirtURL.
func credentials(libvirtURL, username, password string) (string, string, error) {
	return readCredentials(libvirtURL, username, password, secretAccessKeyPath, secretSecretKeyPath)
}

func readCredentials(libvirtURL, username, password, userFile, passFile string) (string, string, error) {
	user := username
	pass := password

	if strings.TrimSpace(user) == "" {
		if b, err := os.ReadFile(userFile); err == nil {
			user = strings.TrimSpace(string(b))
		} else if !os.IsNotExist(err) {
			return "", "", fmt.Errorf("read vSphere username: %w", err)
		}
	}
	if strings.TrimSpace(user) == "" && libvirtURL != "" {
		u, parseErr := url.Parse(libvirtURL)
		if parseErr != nil {
			return "", "", parseErr
		}
		if u.User != nil {
			user = u.User.Username()
		}
	}

	if strings.TrimSpace(pass) == "" {
		if b, err := os.ReadFile(passFile); err == nil {
			pass = strings.TrimSpace(string(b))
		} else if !os.IsNotExist(err) {
			return "", "", fmt.Errorf("read vSphere password: %w", err)
		}
	}

	if strings.TrimSpace(user) == "" || strings.TrimSpace(pass) == "" {
		return "", "", fmt.Errorf("vSphere credentials are empty")
	}
	return user, pass, nil
}

// connectSDK dials the vCenter SOAP API. TLS uses CA policy; optional fingerprint
// is a govmomi fallback when CA verification fails. ESXi NFC downloads reuse this
// client and inherit ESXi thumbprints registered during lease.Wait().
func connectSDK(ctx context.Context, sdk *url.URL, policy v2vtls.Policy, fingerprint, libvirtURL, username, password string) (*govmomi.Client, error) {
	user, pass, err := credentials(libvirtURL, username, password)
	if err != nil {
		return nil, err
	}
	sdk.User = url.UserPassword(user, pass)

	tlsCfg, err := v2vtls.VCenterConfig(policy)
	if err != nil {
		return nil, err
	}

	soapClient := soap.NewClient(sdk, policy.Mode == v2vtls.ModeInsecure)
	soapClient.DefaultTransport().TLSClientConfig = tlsCfg
	if fingerprint != "" {
		soapClient.SetThumbprint(sdk.Host, fingerprint)
	}

	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, fmt.Errorf("connect to vSphere: %w", err)
	}

	c := &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}
	if err = c.Login(ctx, sdk.User); err != nil {
		return nil, fmt.Errorf("connect to vSphere: %w", err)
	}
	return c, nil
}

func connect(ctx context.Context, libvirtURL string, fingerprint string) (*govmomi.Client, error) {
	sdk, urlInsecure, err := sdkURL(libvirtURL)
	if err != nil {
		return nil, err
	}
	policy, err := v2vtls.ForkliftTLS(urlInsecure)
	if err != nil {
		return nil, err
	}
	return connectSDK(ctx, sdk, policy, fingerprint, libvirtURL, "", "")
}

// ConnectHost connects to https://host/sdk. Username and password take
// precedence over Forklift secret files when non-empty.
func ConnectHost(ctx context.Context, host string, policy v2vtls.Policy, fingerprint, username, password string) (*govmomi.Client, error) {
	sdk := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/sdk",
	}
	return connectSDK(ctx, sdk, policy, fingerprint, "", username, password)
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
