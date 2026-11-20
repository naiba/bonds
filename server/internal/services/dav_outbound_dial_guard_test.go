package services

import "testing"

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
