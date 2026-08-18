//go:build unix

package staticip

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestMacToIPLine(t *testing.T) {
	cases := []struct {
		name string
		sip  types.StaticIP
		want string
	}{
		{
			name: "default prefix",
			sip:  types.StaticIP{MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.5", Gateway: "10.0.0.1"},
			want: "52:54:00:aa:bb:cc:ip:10.0.0.5,10.0.0.1,24,",
		},
		{
			name: "netmask 16",
			sip: types.StaticIP{
				MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.5", Gateway: "10.0.0.1",
				Netmask: "255.255.0.0", DNS: []string{"8.8.8.8", "8.8.4.4"},
			},
			want: "52:54:00:aa:bb:cc:ip:10.0.0.5,10.0.0.1,16,8.8.8.8,8.8.4.4",
		},
		{
			name: "invalid netmask keeps 24",
			sip: types.StaticIP{
				MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.5", Gateway: "10.0.0.1",
				Netmask: "invalid",
			},
			want: "52:54:00:aa:bb:cc:ip:10.0.0.5,10.0.0.1,24,",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MacToIPLine(&tc.sip); got != tc.want {
				t.Errorf("MacToIPLine = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNetmaskPrefix(t *testing.T) {
	cases := []struct {
		mask    string
		want    string
		wantErr bool
	}{
		{"255.255.255.0", "24", false},
		{"255.255.0.0", "16", false},
		{"invalid", "24", true},
		{"255.255", "24", true},
	}
	for _, tc := range cases {
		got, err := netmaskPrefix(tc.mask)
		if got != tc.want {
			t.Errorf("netmaskPrefix(%q) = %q, want %q", tc.mask, got, tc.want)
		}
		if (err != nil) != tc.wantErr {
			t.Errorf("netmaskPrefix(%q) err = %v, wantErr %v", tc.mask, err, tc.wantErr)
		}
	}
}

func TestWriteMacToIPEmpty(t *testing.T) {
	root := t.TempDir()
	if err := WriteMacToIP(root, nil); err != nil {
		t.Fatalf("WriteMacToIP: %v", err)
	}
	path := filepath.Join(root, "tmp", "macToIP")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected no macToIP file when ips empty")
	}
}

func TestWriteMacToIP(t *testing.T) {
	root := t.TempDir()
	ips := []types.StaticIP{
		{MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.5", Gateway: "10.0.0.1", Netmask: "255.255.255.0"},
	}
	if err := WriteMacToIP(root, ips); err != nil {
		t.Fatalf("WriteMacToIP: %v", err)
	}
	path := filepath.Join(root, "tmp", "macToIP")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := "52:54:00:aa:bb:cc:ip:10.0.0.5,10.0.0.1,24,\n"
	if string(data) != want {
		t.Errorf("macToIP = %q, want %q", data, want)
	}
}

func TestFirstbootCommands(t *testing.T) {
	cmds := FirstbootCommands()
	if len(cmds) == 0 {
		t.Fatal("expected commands")
	}
	joined := strings.Join(cmds, "\n")
	if !strings.Contains(joined, "/tmp/macToIP") {
		t.Errorf("commands missing /tmp/macToIP: %v", cmds)
	}
}
