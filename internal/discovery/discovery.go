package discovery

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DiscoveryConfig struct {
	SSHDir         string
	KnownHostsPath string
	SSHConfigPath  string
}

func DefaultConfig() *DiscoveryConfig {
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	return &DiscoveryConfig{
		SSHDir:         sshDir,
		KnownHostsPath: filepath.Join(sshDir, "known_hosts"),
		SSHConfigPath:  filepath.Join(sshDir, "config"),
	}
}

type DiscoveredHost struct {
	Hostname  string
	IPAddress string
	Port      int
	Source    string
	User      string
}

type Discoverer struct {
	config *DiscoveryConfig
}

func New(config *DiscoveryConfig) *Discoverer {
	return &Discoverer{config: config}
}

func (d *Discoverer) DiscoverAll() ([]*DiscoveredHost, error) {
	var hosts []*DiscoveredHost

	knownHosts, err := d.discoverKnownHosts()
	if err != nil {
		return nil, fmt.Errorf("known_hosts discovery failed: %w", err)
	}
	hosts = append(hosts, knownHosts...)

	sshConfigHosts, err := d.discoverSSHConfig()
	if err != nil {
		return nil, fmt.Errorf("ssh config discovery failed: %w", err)
	}
	hosts = append(hosts, sshConfigHosts...)

	return hosts, nil
}

func (d *Discoverer) discoverKnownHosts() ([]*DiscoveredHost, error) {
	file, err := os.Open(d.config.KnownHostsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var hosts []*DiscoveredHost
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		host := parseKnownHostsLine(line)
		if host != nil {
			hosts = append(hosts, host)
		}
	}

	return hosts, scanner.Err()
}

func parseKnownHostsLine(line string) *DiscoveredHost {
	parts := strings.Fields(line)
	if len(parts) < 1 {
		return nil
	}

	hostPattern := parts[0]

	if strings.Contains(hostPattern, ",") {
		hostPattern = strings.Split(hostPattern, ",")[0]
	}

	if strings.Contains(hostPattern, "|") {
		return nil
	}

	var hostname string
	var ip string
	port := 22

	switch {
	case strings.Contains(hostPattern, "["):
		re := regexp.MustCompile(`\[([^\]]+)\]:?(\d+)?`)
		matches := re.FindStringSubmatch(hostPattern)
		if len(matches) >= 2 {
			hostname = matches[1]
			if len(matches) >= 3 && matches[2] != "" {
				if _, err := fmt.Sscanf(matches[2], "%d", &port); err != nil {
					port = 22
				}
			}
		}
	case strings.Contains(hostPattern, ":") && net.ParseIP(hostPattern) != nil:
		ip = hostPattern
		hostname = parts[0]
	case strings.Contains(hostPattern, "@"):
		hostname = strings.Split(hostPattern, "@")[1]
	default:
		hostname = hostPattern
	}

	if ip == "" && net.ParseIP(hostname) != nil {
		ip = hostname
	}

	if hostname == "" {
		return nil
	}

	return &DiscoveredHost{
		Hostname:  hostname,
		IPAddress: ip,
		Port:      port,
		Source:    "known_hosts",
	}
}

func (d *Discoverer) discoverSSHConfig() ([]*DiscoveredHost, error) {
	file, err := os.Open(d.config.SSHConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var hosts []*DiscoveredHost
	scanner := bufio.NewScanner(file)
	var currentHost string
	port := 22
	var ip string
	var user string

	hostPattern := regexp.MustCompile(`^Host\s+(.+)$`)
	hostVarPattern := regexp.MustCompile(`^(HostName|Port|User|IdentityFile)\s+(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if hostMatches := hostPattern.FindStringSubmatch(line); hostMatches != nil {
			if currentHost != "" {
				hosts = append(hosts, &DiscoveredHost{
					Hostname:  currentHost,
					IPAddress: ip,
					Port:      port,
					Source:    "ssh_config",
					User:      user,
				})
			}
			currentHost = hostMatches[1]
			if currentHost == "*" {
				currentHost = ""
			}
			port = 22
			ip = ""
			user = ""
			continue
		}

		if currentHost == "" {
			continue
		}

		if varMatches := hostVarPattern.FindStringSubmatch(line); varMatches != nil {
			switch varMatches[1] {
			case "HostName":
				ip = strings.Trim(varMatches[2], "\" ")
			case "Port":
				fmt.Sscanf(varMatches[2], "%d", &port)
			case "User":
				user = strings.Trim(varMatches[2], "\" ")
			}
		}
	}

	if currentHost != "" {
		hosts = append(hosts, &DiscoveredHost{
			Hostname:  currentHost,
			IPAddress: ip,
			Port:      port,
			Source:    "ssh_config",
			User:      user,
		})
	}

	return hosts, scanner.Err()
}
