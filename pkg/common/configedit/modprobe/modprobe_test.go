package modprobe

import (
	"strings"
	"testing"
)

const testModprobe = `# Custom modprobe config
alias eth0 e1000
alias wlan0 ath9k
options snd_hda_intel power_save=1
blacklist nouveau
`

func TestParse(t *testing.T) {
	c := Parse(testModprobe)
	aliases := c.Aliases()
	if len(aliases) != 2 {
		t.Fatalf("got %d aliases, want 2", len(aliases))
	}
	if aliases["eth0"] != "e1000" {
		t.Errorf("alias eth0 = %q, want e1000", aliases["eth0"])
	}
	if aliases["wlan0"] != "ath9k" {
		t.Errorf("alias wlan0 = %q, want ath9k", aliases["wlan0"])
	}
}

func TestAddAlias(t *testing.T) {
	c := Parse(testModprobe)
	c.AddAlias("usb0", "cdc_ether")
	aliases := c.Aliases()
	if aliases["usb0"] != "cdc_ether" {
		t.Errorf("alias usb0 = %q, want cdc_ether", aliases["usb0"])
	}
	if len(aliases) != 3 {
		t.Errorf("got %d aliases, want 3", len(aliases))
	}
}

func TestAddAliasReplace(t *testing.T) {
	c := Parse(testModprobe)
	c.AddAlias("eth0", "virtio_net")
	aliases := c.Aliases()
	if aliases["eth0"] != "virtio_net" {
		t.Errorf("alias eth0 = %q, want virtio_net", aliases["eth0"])
	}
	if len(aliases) != 2 {
		t.Errorf("got %d aliases after replace, want 2", len(aliases))
	}
}

func TestAddOption(t *testing.T) {
	c := Parse(testModprobe)
	c.AddOption("virtio_blk", "num_queues=4")
	out := c.String()
	if !strings.Contains(out, "options virtio_blk num_queues=4") {
		t.Errorf("output missing added option line")
	}
}

func TestAddBlacklist(t *testing.T) {
	c := Parse(testModprobe)
	c.AddBlacklist("floppy")
	out := c.String()
	if !strings.Contains(out, "blacklist floppy") {
		t.Errorf("output missing blacklist floppy")
	}
}

func TestString(t *testing.T) {
	c := Parse(testModprobe)
	out := c.String()
	if !strings.Contains(out, "alias eth0 e1000") {
		t.Error("output should contain alias line")
	}
	if !strings.Contains(out, "# Custom modprobe config") {
		t.Error("output should preserve comments")
	}
	if !strings.Contains(out, "blacklist nouveau") {
		t.Error("output should contain blacklist line")
	}
}
