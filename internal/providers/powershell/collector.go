package powershell

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/engine"
	"github.com/dwg7318/codex-network-doctor/internal/evidence"
)

type Runner interface {
	Run(context.Context, string, ...string) (stdout []byte, stderr []byte, exitCode int, err error)
}

type Script struct {
	ID          string        `json:"id"`
	Category    string        `json:"category"`
	Description string        `json:"description"`
	Body        string        `json:"script"`
	Timeout     time.Duration `json:"-"`
	TimeoutMS   int64         `json:"timeout_ms"`
	Privilege   string        `json:"privilege"`
	Mode        string        `json:"mode"`
	MediaType   string        `json:"media_type"`
	Extension   string        `json:"extension"`
}

type Collector struct {
	binary string
	script Script
	runner Runner
}

func NewCollector(binary string, script Script, runner Runner) Collector {
	if script.Mode == "" {
		script.Mode = "passive"
	}
	if script.MediaType == "" {
		script.MediaType = "application/json"
	}
	if script.Extension == "" {
		script.Extension = ".json"
	}
	return Collector{binary: binary, script: script, runner: runner}
}

func (c Collector) Definition() engine.Definition {
	return engine.Definition{
		ID:          c.script.ID,
		Category:    c.script.Category,
		Description: c.script.Description,
		Provider:    "windows-powershell",
		Timeout:     c.script.Timeout,
		Privilege:   c.script.Privilege,
		Mode:        c.script.Mode,
	}
}

func (c Collector) Collect(ctx context.Context, store *evidence.Store) ([]evidence.Artifact, error) {
	stdout, stderr, exitCode, runErr := c.runner.Run(
		ctx,
		c.binary,
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		c.script.Body,
	)

	artifacts := make([]evidence.Artifact, 0, 2)
	stdoutArtifact, writeErr := store.Write(
		"evidence/windows/"+c.script.ID+c.script.Extension,
		stdout,
		c.script.MediaType,
	)
	if writeErr != nil {
		return artifacts, writeErr
	}
	artifacts = append(artifacts, stdoutArtifact)

	if len(stderr) > 0 {
		stderrArtifact, err := store.Write(
			"logs/windows/"+c.script.ID+".stderr.txt",
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
		return artifacts, fmt.Errorf("PowerShell collector %q exit code %d: %w", c.script.ID, exitCode, runErr)
	}
	if strings.HasPrefix(c.script.MediaType, "application/json") && !json.Valid(stdout) {
		return artifacts, fmt.Errorf("PowerShell collector %q returned invalid JSON", c.script.ID)
	}
	return artifacts, nil
}

func DefaultScripts() []Script {
	const standardTimeout = 15 * time.Second
	const longerTimeout = 30 * time.Second
	const prefix = "$ErrorActionPreference='Stop'; [Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false); $OutputEncoding=[Console]::OutputEncoding; "
	return []Script{
		jsonScript(
			"system_info", "system", "Windows host, operating system, hardware, and timezone summary.", standardTimeout,
			prefix+"$os=Get-CimInstance Win32_OperatingSystem; $cs=Get-CimInstance Win32_ComputerSystem; $tz=Get-TimeZone; [pscustomobject]@{computer_name=$env:COMPUTERNAME; os_caption=$os.Caption; os_version=$os.Version; os_build=$os.BuildNumber; os_architecture=$os.OSArchitecture; last_boot_time=$os.LastBootUpTime; manufacturer=$cs.Manufacturer; model=$cs.Model; domain=$cs.Domain; total_physical_memory=$cs.TotalPhysicalMemory; timezone_id=$tz.Id; timezone_display_name=$tz.DisplayName} | ConvertTo-Json -Depth 4 -Compress",
		),
		jsonScript(
			"net_adapters", "adapter", "Network adapter state, MAC address, link speed, and driver description.", standardTimeout,
			prefix+"$items=@(Get-NetAdapter -IncludeHidden | Select-Object Name,InterfaceDescription,InterfaceGuid,ifIndex,Status,MacAddress,LinkSpeed,MediaType,PhysicalMediaType,DriverInformation); ConvertTo-Json -InputObject $items -Depth 4 -Compress",
		),
		jsonScript(
			"ip_configuration", "adapter", "Assigned addresses, gateways, profiles, and DNS servers per interface.", standardTimeout,
			prefix+"$items=@(Get-NetIPConfiguration -All | ForEach-Object { [pscustomobject]@{interface_alias=$_.InterfaceAlias; interface_index=$_.InterfaceIndex; interface_description=$_.InterfaceDescription; net_profile_name=$_.NetProfile.Name; ipv4_addresses=@($_.IPv4Address | ForEach-Object { [pscustomobject]@{address=$_.IPAddress; prefix_length=$_.PrefixLength} }); ipv6_addresses=@($_.IPv6Address | ForEach-Object { [pscustomobject]@{address=$_.IPAddress; prefix_length=$_.PrefixLength} }); ipv4_gateways=@($_.IPv4DefaultGateway | Select-Object -ExpandProperty NextHop); ipv6_gateways=@($_.IPv6DefaultGateway | Select-Object -ExpandProperty NextHop); dns_servers=@($_.DNSServer.ServerAddresses) } }); ConvertTo-Json -InputObject $items -Depth 8 -Compress",
		),
		jsonScript(
			"routes", "route", "IPv4 and IPv6 route table.", standardTimeout,
			prefix+"$items=@(Get-NetRoute | Select-Object DestinationPrefix,NextHop,InterfaceAlias,InterfaceIndex,RouteMetric,InterfaceMetric,Protocol,Publish,State,AddressFamily); ConvertTo-Json -InputObject $items -Depth 4 -Compress",
		),
		jsonScript(
			"neighbors", "route", "ARP and IPv6 neighbor cache.", standardTimeout,
			prefix+"$items=@(Get-NetNeighbor | Select-Object IPAddress,LinkLayerAddress,State,InterfaceAlias,InterfaceIndex,AddressFamily,PolicyStore); ConvertTo-Json -InputObject $items -Depth 4 -Compress",
		),
		jsonScript(
			"dns_servers", "dns", "DNS server configuration per interface.", standardTimeout,
			prefix+"$items=@(Get-DnsClientServerAddress | Select-Object InterfaceAlias,InterfaceIndex,AddressFamily,ServerAddresses); ConvertTo-Json -InputObject $items -Depth 5 -Compress",
		),
		jsonScript(
			"dns_cache", "dns", "Current Windows DNS client cache.", standardTimeout,
			prefix+"$items=@(Get-DnsClientCache | Select-Object Entry,Name,Data,Type,Status,Section,TimeToLive); ConvertTo-Json -InputObject $items -Depth 4 -Compress",
		),
		jsonScript(
			"tcp_connections", "socket", "Current TCP endpoints and owning process IDs.", standardTimeout,
			prefix+"$items=@(Get-NetTCPConnection | Select-Object LocalAddress,LocalPort,RemoteAddress,RemotePort,State,OwningProcess,AppliedSetting,CreationTime); ConvertTo-Json -InputObject $items -Depth 4 -Compress",
		),
		jsonScript(
			"udp_endpoints", "socket", "Current UDP endpoints and owning process IDs.", standardTimeout,
			prefix+"$items=@(Get-NetUDPEndpoint | Select-Object LocalAddress,LocalPort,OwningProcess,CreationTime); ConvertTo-Json -InputObject $items -Depth 4 -Compress",
		),
		{
			ID: "winhttp_proxy", Category: "proxy", Description: "Raw WinHTTP proxy configuration.",
			Body:    prefix + "$output=@(& netsh.exe winhttp show proxy 2>&1); if ($LASTEXITCODE -ne 0) { throw \"netsh exited with $LASTEXITCODE\" }; $output",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
			MediaType: "text/plain; charset=utf-8", Extension: ".txt",
		},
		jsonScript(
			"user_proxy", "proxy", "Current-user Internet Settings proxy and PAC configuration.", standardTimeout,
			prefix+"$value=Get-ItemProperty -LiteralPath 'HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings'; [pscustomobject]@{proxy_enable=$value.ProxyEnable; proxy_server=$value.ProxyServer; proxy_override=$value.ProxyOverride; auto_config_url=$value.AutoConfigURL; auto_detect=$value.AutoDetect} | ConvertTo-Json -Depth 3 -Compress",
		),
		jsonScript(
			"firewall_profiles", "firewall", "Windows Firewall profile state and default actions.", longerTimeout,
			prefix+"$items=@(Get-NetFirewallProfile | Select-Object Name,Enabled,DefaultInboundAction,DefaultOutboundAction,AllowInboundRules,AllowLocalFirewallRules,AllowLocalIPsecRules,NotifyOnListen,LogFileName,LogMaxSizeKilobytes,LogAllowed,LogBlocked,LogIgnored); ConvertTo-Json -InputObject $items -Depth 4 -Compress",
		),
		jsonScript(
			"processes", "process", "Process identity and executable paths without command lines or environment variables.", standardTimeout,
			prefix+"$items=@(Get-Process | Select-Object Id,ProcessName,Path,StartTime); ConvertTo-Json -InputObject $items -Depth 4 -Compress",
		),
		jsonScript(
			"certificates", "tls", "Public certificate metadata; private key material is never exported.", longerTimeout,
			prefix+"$items=@(); $definitions=@(@{name='My';location='CurrentUser'},@{name='Root';location='LocalMachine'}); foreach($definition in $definitions) { $location=[System.Enum]::Parse([System.Security.Cryptography.X509Certificates.StoreLocation],$definition.location); $store=[System.Security.Cryptography.X509Certificates.X509Store]::new($definition.name,$location); try { $store.Open([System.Security.Cryptography.X509Certificates.OpenFlags]::ReadOnly); foreach($certificate in $store.Certificates) { $items += [pscustomobject]@{store_name=$definition.name; store_location=$definition.location; subject=$certificate.Subject; issuer=$certificate.Issuer; thumbprint=$certificate.Thumbprint; serial_number=$certificate.SerialNumber; not_before=$certificate.NotBefore; not_after=$certificate.NotAfter; has_private_key=$certificate.HasPrivateKey; signature_algorithm=$certificate.SignatureAlgorithm.Value; public_key_algorithm=$certificate.PublicKey.Oid.Value; dns_name=$certificate.GetNameInfo([System.Security.Cryptography.X509Certificates.X509NameType]::DnsName,$false)} } } finally { $store.Close() } }; ConvertTo-Json -InputObject $items -Depth 6 -Compress",
		),
		{
			ID: "hosts_file", Category: "dns", Description: "Raw Windows hosts file.",
			Body:    prefix + "Get-Content -LiteralPath (Join-Path $env:SystemRoot 'System32\\drivers\\etc\\hosts') -Raw",
			Timeout: standardTimeout, TimeoutMS: standardTimeout.Milliseconds(), Privilege: "user", Mode: "passive",
			MediaType: "text/plain; charset=utf-8", Extension: ".txt",
		},
	}
}

func jsonScript(id, category, description string, timeout time.Duration, body string) Script {
	return Script{
		ID: id, Category: category, Description: description, Body: body,
		Timeout: timeout, TimeoutMS: timeout.Milliseconds(), Privilege: "user", Mode: "passive",
		MediaType: "application/json", Extension: ".json",
	}
}
