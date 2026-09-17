package api

import (
	"fmt"
	"net/netip"
	"strings"
)

// ValidateServerRoutes parses raw as a comma-separated list of CIDR
// networks - the networks behind a client, added to the server's
// AllowedIPs for that peer on top of the client's own /32 - and reports
// every problem found.
//
// It does not check for overlaps between clients: a route already assigned
// to another peer is left to the administrator to catch, not enforced here.
//
// An empty raw is valid and means "no additional networks" - the server
// then routes only the client's own address to it.
func ValidateServerRoutes(raw string) (problems []string) {
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(part)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%q is not a valid CIDR network", part))
			continue
		}
		if prefix.Bits() == 0 {
			problems = append(problems, fmt.Sprintf(
				"%q would route all traffic through this client; AllowedIPs cannot express that as a peer-specific route", part))
		}
	}
	return problems
}

// NormalizeServerRoutes trims whitespace, drops empty elements and removes
// exact duplicates. Call it only after ValidateServerRoutes has returned no
// problems - it does not itself check that raw is valid.
func NormalizeServerRoutes(raw string) string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return strings.Join(out, ", ")
}
