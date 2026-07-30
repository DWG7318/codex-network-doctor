package osquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/engine"
	"github.com/dwg7318/codex-network-doctor/internal/evidence"
)

type Runner interface {
	Run(context.Context, string, ...string) (stdout []byte, stderr []byte, exitCode int, err error)
}

type Query struct {
	ID          string        `json:"id"`
	Category    string        `json:"category"`
	Description string        `json:"description"`
	SQL         string        `json:"sql"`
	Timeout     time.Duration `json:"-"`
	TimeoutMS   int64         `json:"timeout_ms"`
	Privilege   string        `json:"privilege"`
	Mode        string        `json:"mode"`
}

type Collector struct {
	binary string
	query  Query
	runner Runner
}

func NewCollector(binary string, query Query, runner Runner) Collector {
	if query.Mode == "" {
		query.Mode = "passive"
	}
	return Collector{binary: binary, query: query, runner: runner}
}

func (c Collector) Definition() engine.Definition {
	return engine.Definition{
		ID:          c.query.ID,
		Category:    c.query.Category,
		Description: c.query.Description,
		Provider:    "osquery",
		Timeout:     c.query.Timeout,
		Privilege:   c.query.Privilege,
		Mode:        c.query.Mode,
	}
}

func (c Collector) Collect(ctx context.Context, store *evidence.Store) ([]evidence.Artifact, error) {
	stdout, stderr, exitCode, runErr := c.runner.Run(
		ctx,
		c.binary,
		"--json",
		"--disable_events=true",
		"--disable_extensions=true",
		"--disable_logging=true",
		"--logger_stderr=false",
		"--utc=true",
		c.query.SQL,
	)

	artifacts := make([]evidence.Artifact, 0, 2)
	stdoutArtifact, writeErr := store.Write(
		"evidence/osquery/"+c.query.ID+".json",
		stdout,
		"application/json",
	)
	if writeErr != nil {
		return artifacts, writeErr
	}
	artifacts = append(artifacts, stdoutArtifact)

	if len(stderr) > 0 {
		stderrArtifact, err := store.Write(
			"logs/osquery/"+c.query.ID+".stderr.txt",
			stderr,
			"text/plain; charset=utf-8",
		)
		if err != nil {
			return artifacts, err
		}
		artifacts = append(artifacts, stderrArtifact)
	}

	if runErr != nil || exitCode != 0 {
		if runErr == nil {
			runErr = errors.New("process returned a non-zero exit code")
		}
		return artifacts, fmt.Errorf("osquery query %q exit code %d: %w", c.query.ID, exitCode, runErr)
	}
	if !json.Valid(stdout) {
		return artifacts, fmt.Errorf("osquery query %q returned invalid JSON", c.query.ID)
	}
	return artifacts, nil
}

func DefaultQueries() []Query {
	const standardTimeout = 15 * time.Second
	return []Query{
		{
			ID: "system_info", Category: "system", Description: "Host identity and hardware summary.",
			SQL:     "SELECT hostname, uuid, computer_name, local_hostname, cpu_type, cpu_subtype, cpu_brand, physical_memory, hardware_vendor, hardware_model, hardware_serial FROM system_info;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "os_version", Category: "system", Description: "Windows version and build metadata.",
			SQL:     "SELECT name, version, major, minor, patch, build, platform, platform_like, codename, arch FROM os_version;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "interface_addresses", Category: "adapter", Description: "IP addresses assigned to network interfaces.",
			SQL:     "SELECT interface, address, mask, broadcast, point_to_point, type, friendly_name FROM interface_addresses;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "interface_details", Category: "adapter", Description: "Interface state, MAC, MTU, metrics, and Windows DNS details.",
			SQL:     "SELECT * FROM interface_details;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "routes", Category: "route", Description: "IPv4 and IPv6 routing table.",
			SQL:     "SELECT destination, netmask, gateway, source, flags, interface, mtu, metric, type FROM routes;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "arp_cache", Category: "route", Description: "ARP and neighbor cache entries.",
			SQL:     "SELECT address, mac, interface, permanent FROM arp_cache;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "dns_configuration", Category: "dns", Description: "Windows DNS host, domain, suffix search order, and server search order per interface.",
			SQL:     "SELECT interface, friendly_name, dns_domain, dns_domain_suffix_search_order, dns_host_name, dns_server_search_order FROM interface_details;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "process_open_sockets", Category: "socket", Description: "Current process-owned TCP and UDP sockets.",
			SQL:     "SELECT pid, family, protocol, local_address, remote_address, local_port, remote_port, path, state FROM process_open_sockets;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "listening_ports", Category: "socket", Description: "Listening network endpoints and owning processes.",
			SQL:     "SELECT pid, port, protocol, family, address, fd, socket, path FROM listening_ports;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "processes", Category: "process", Description: "Process identity without command-line arguments or environment data.",
			SQL:     "SELECT pid, name, path, parent, start_time FROM processes;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "services", Category: "process", Description: "Installed Windows service state and image paths.",
			SQL:     "SELECT name, display_name, status, start_type, path, module_path, service_exit_code, user_account FROM services;",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "certificates", Category: "tls", Description: "Public certificate metadata from Windows certificate stores.",
			SQL:     "SELECT common_name, subject, issuer, ca, self_signed, not_valid_before, not_valid_after, signing_algorithm, key_algorithm, key_strength, path, serial, sha1 FROM certificates;",
			Timeout: 30 * time.Second, TimeoutMS: (30 * time.Second).Milliseconds(), Privilege: "user", Mode: "passive",
		},
		{
			ID: "windows_firewall_rules", Category: "firewall", Description: "Configured Windows Firewall rules.",
			SQL:     "SELECT * FROM windows_firewall_rules;",
			Timeout: 30 * time.Second, TimeoutMS: (30 * time.Second).Milliseconds(), Privilege: "user", Mode: "passive",
		},
	}
}
