package operator

import (
	"os"
	"path/filepath"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlatformManifests(t *testing.T) {
	cases := []struct {
		name              string
		platformName      string
		lbType            configv1.PlatformLoadBalancerType
		vipManagement     string
		expectKeepalived  bool
		expectFRRK8s      bool
		expectCoredns     bool
		expectKubeVIPAPI  bool
	}{
		{
			name:              "baremetal default LB, no BGP",
			platformName:      "baremetal",
			lbType:            configv1.LoadBalancerTypeOpenShiftManagedDefault,
			vipManagement:     "",
			expectKeepalived:  true,
			expectFRRK8s:      false,
			expectCoredns:     true,
			expectKubeVIPAPI:  false,
		},
		{
			name:              "baremetal default LB, BGP enabled",
			platformName:      "baremetal",
			lbType:            configv1.LoadBalancerTypeOpenShiftManagedDefault,
			vipManagement:     "BGP",
			expectKeepalived:  false,
			expectFRRK8s:      true,
			expectCoredns:     true,
			expectKubeVIPAPI:  true,
		},
		{
			name:              "baremetal user-managed LB, no BGP",
			platformName:      "baremetal",
			lbType:            configv1.LoadBalancerTypeUserManaged,
			vipManagement:     "",
			expectKeepalived:  false,
			expectFRRK8s:      false,
			expectCoredns:     true,
			expectKubeVIPAPI:  false,
		},
		{
			name:              "baremetal user-managed LB, BGP enabled",
			platformName:      "baremetal",
			lbType:            configv1.LoadBalancerTypeUserManaged,
			vipManagement:     "BGP",
			expectKeepalived:  false,
			expectFRRK8s:      false,
			expectCoredns:     true,
			expectKubeVIPAPI:  false,
		},
		{
			name:              "baremetal empty LB type, no BGP",
			platformName:      "baremetal",
			lbType:            "",
			vipManagement:     "",
			expectKeepalived:  true,
			expectFRRK8s:      false,
			expectCoredns:     true,
			expectKubeVIPAPI:  false,
		},
		{
			name:              "baremetal empty LB type, BGP enabled",
			platformName:      "baremetal",
			lbType:            "",
			vipManagement:     "BGP",
			expectKeepalived:  false,
			expectFRRK8s:      true,
			expectCoredns:     true,
			expectKubeVIPAPI:  true,
		},
		{
			name:              "openstack default LB, no BGP",
			platformName:      "openstack",
			lbType:            configv1.LoadBalancerTypeOpenShiftManagedDefault,
			vipManagement:     "",
			expectKeepalived:  true,
			expectFRRK8s:      false,
			expectCoredns:     true,
			expectKubeVIPAPI:  false,
		},
		{
			name:              "gcp cloud platform",
			platformName:      "gcp",
			lbType:            configv1.LoadBalancerTypeUserManaged,
			vipManagement:     "",
			expectKeepalived:  false,
			expectFRRK8s:      false,
			expectCoredns:     true,
			expectKubeVIPAPI:  false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := getPlatformManifests(nil, c.platformName, c.lbType, c.vipManagement)

			hasKeepalived := false
			hasFRRK8s := false
			hasCoredns := false
			hasKubeVIPAPI := false
			for _, m := range result {
				if m.name == "manifests/on-prem/keepalived.yaml" {
					hasKeepalived = true
				}
				if m.name == "manifests/on-prem/0000-frr-k8s.yaml" {
					hasFRRK8s = true
				}
				if m.name == "manifests/on-prem/coredns.yaml" || m.name == "manifests/cloud-platform-alt-dns/coredns.yaml" {
					hasCoredns = true
				}
				if m.name == "manifests/on-prem/0010-kube-vip-api.yaml" {
					hasKubeVIPAPI = true
				}
			}

			assert.Equal(t, c.expectKeepalived, hasKeepalived, "keepalived manifest presence")
			assert.Equal(t, c.expectFRRK8s, hasFRRK8s, "frr-k8s manifest presence")
			assert.Equal(t, c.expectCoredns, hasCoredns, "coredns manifest presence")
			assert.Equal(t, c.expectKubeVIPAPI, hasKubeVIPAPI, "kube-vip-api manifest presence")
		})
	}
}

func TestFillBGPVIPConfig(t *testing.T) {
	dir := t.TempDir()
	cmYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: bgp-vip-config
  namespace: openshift-network-operator
data:
  config.json: '{"localASN":64512,"defaultPeers":[{"peerAddress":"192.168.111.1","peerASN":64513}],"apiVIPs":["192.168.111.5"],"ingressVIPs":["192.168.111.4"]}'
`
	path := filepath.Join(dir, "bgp-vip-config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cmYAML), 0o644))

	deps := &BootstrapDependencies{}
	require.NoError(t, deps.fillBGPVIPConfig(path), "fillBGPVIPConfig")
	assert.Contains(t, deps.BGPVIPPeersJSON, `"defaultPeers"`, "BGPVIPPeersJSON missing defaultPeers")

	// Missing file is tolerated (optional dependency).
	deps2 := &BootstrapDependencies{}
	require.NoError(t, deps2.fillBGPVIPConfig(filepath.Join(dir, "nonexistent.yaml")), "missing file must not error")
	assert.Empty(t, deps2.BGPVIPPeersJSON, "expected empty for missing file")
}
