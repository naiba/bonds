package services

import (
	"context"
	"strings"
	"testing"
)

// Link-local addresses are where cloud providers serve instance credentials.
// No address book lives there, while LAN and loopback servers are legitimate
// targets for a self-hosted instance and must stay reachable.
func TestRejectLinkLocalDial(t *testing.T) {
	cases := []struct {
		address string
		blocked bool
	}{
		{"169.254.169.254:80", true},
		{"169.254.10.20:443", true},
		{"[fe80::1]:443", true},
		{"[fe80::1%eth0]:443", true},
		// Metadata endpoints that are not link-local.
		{"[fd00:ec2::254]:80", true},
		{"100.100.100.200:80", true},
		{"192.0.0.192:80", true},
		{"100.100.100.201:80", false},
		{"192.168.1.20:443", false},
		{"10.0.0.5:8443", false},
		{"127.0.0.1:5232", false},
		{"[::1]:5232", false},
		{"203.0.113.10:443", false},
		{"[2001:db8::10]:443", false},
	}
	for _, tc := range cases {
		err := rejectLinkLocalDial("tcp", tc.address, nil)
		if tc.blocked && err == nil {
			t.Errorf("dial to %s was allowed, want it blocked", tc.address)
		}
		if !tc.blocked && err != nil {
			t.Errorf("dial to %s was blocked: %v", tc.address, err)
		}
	}
}

// The guard is only worth something while the real client uses it.
func TestDefaultCardDAVClient_RefusesInstanceMetadataAddress(t *testing.T) {
	client, err := (&DefaultCardDAVClientFactory{}).NewClient("http://169.254.169.254/latest/meta-data/", "user", "password", DavTLSConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.FindCurrentUserPrincipal(context.Background())

	if err == nil || !strings.Contains(err.Error(), "refusing to connect") {
		t.Fatalf("request to the metadata address: error = %v, want the dial to be refused", err)
	}
}
