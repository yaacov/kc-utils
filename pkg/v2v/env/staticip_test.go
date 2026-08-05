package env

import "testing"

func TestParseStaticIPs(t *testing.T) {
	raw := "52:54:00:aa:bb:cc:ip:192.168.1.10,192.168.1.1,24,8.8.8.8,8.8.4.4"
	ips, err := ParseStaticIPs(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 1 {
		t.Fatalf("got %d ips", len(ips))
	}
	if ips[0].MAC != "52:54:00:aa:bb:cc" || ips[0].IP != "192.168.1.10" {
		t.Fatalf("unexpected ip: %+v", ips[0])
	}
	if ips[0].Gateway != "192.168.1.1" || ips[0].Netmask != "255.255.255.0" {
		t.Fatalf("gateway/netmask: %+v", ips[0])
	}
	if len(ips[0].DNS) != 2 {
		t.Fatalf("dns = %v", ips[0].DNS)
	}
}

func TestNormalizeSourceType(t *testing.T) {
	if NormalizeSourceType("vSphere") != "vsphere" {
		t.Fatal("expected lowercase source type")
	}
	if NormalizeSourceType("nutanix") != "nutanix" {
		t.Fatalf("nutanix = %q", NormalizeSourceType("nutanix"))
	}
	if NormalizeSourceType("nutanix-ahv") != "nutanix" {
		t.Fatalf("nutanix-ahv = %q", NormalizeSourceType("nutanix-ahv"))
	}
}

func TestSourceName(t *testing.T) {
	cfg := &Config{VmName: "myvm"}
	if SourceName(cfg) != "myvm" {
		t.Fatal("wrong source name")
	}
}
