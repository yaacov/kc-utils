package types_test

import (
	"encoding/json"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestConverterOutputJSONRoundTrip(t *testing.T) {
	in := types.ConverterOutput{
		GuestCaps: types.GuestCaps{
			BlockBus:    "virtio",
			NetBus:      "virtio",
			MachineType: "q35",
			Arch:        "x86_64",
		},
		Hypervisor: &types.HypervisorInspection{
			Plugins: []types.HypervisorPluginResult{
				{
					Name:   "vmware",
					Action: types.HypervisorActionCleanup,
					Status: types.HypervisorStatusSucceeded,
				},
				{
					Name:   "kudzu",
					Action: types.HypervisorActionCleanup,
					Status: types.HypervisorStatusFailed,
					Error:  "permission denied",
				},
			},
		},
		Network: &types.NetworkInspection{
			Handler: "networkd",
			Primary: types.NetworkPrimarySystemdNetworkd,
		},
		SELinuxRelabeled: true,
		Warnings:         []string{"example warning"},
		Errors: []types.BlockError{
			{Block: "hypervisor-cleanup/kudzu", Message: "permission denied"},
		},
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out types.ConverterOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Hypervisor == nil || len(out.Hypervisor.Plugins) != 2 {
		t.Fatalf("hypervisor plugins: got %+v", out.Hypervisor)
	}
	if out.Hypervisor.Plugins[1].Error != "permission denied" {
		t.Errorf("plugin error = %q", out.Hypervisor.Plugins[1].Error)
	}
	if out.Network == nil {
		t.Fatal("network inspection missing")
	}
	if out.Network.Handler != "networkd" || out.Network.Primary != types.NetworkPrimarySystemdNetworkd {
		t.Errorf("network = %+v", out.Network)
	}
}

func TestConverterOutputOmitempty(t *testing.T) {
	in := types.ConverterOutput{
		GuestCaps: types.GuestCaps{BlockBus: "virtio", NetBus: "virtio"},
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	for _, key := range []string{"hypervisor", "network", "warnings", "errors"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected %q to be omitted", key)
		}
	}
}
