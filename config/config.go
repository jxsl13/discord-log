package config

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

func NewConfig() Config {
	return Config{}
}

type Config struct {
	DiscordToken   string `koanf:"discord.token" description:"bot or personal discord token"`
	IsBot          bool   `koanf:"user.bot" description:"set to true if the token is a bot token"`
	GraylogAddress string `koanf:"graylog.address" description:"udp gelf endpoint"`
}

func (cfg *Config) Validate() error {
	if cfg.DiscordToken == "" {
		return fmt.Errorf("discord token is empty")
	}

	if cfg.IsBot {
		cfg.DiscordToken = fmt.Sprintf("Bot %s", strings.TrimLeft(cfg.DiscordToken, "Bot "))
	} else {
		cfg.DiscordToken = strings.TrimLeft(cfg.DiscordToken, "Bot ")
	}

	parts := strings.SplitN(cfg.GraylogAddress, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid address, expected host:port: %s", cfg.GraylogAddress)
	}
	host := parts[0]
	port := parts[1]

	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("failed to resolve host: %s: %w", host, err)
	}

	// we try to select ipv4
	var selectedAddr netip.Addr
	for _, ip := range ips {
		addr, err := netip.ParseAddr(ip)
		if err != nil {
			return fmt.Errorf("failed to parse resolved address with port: %s: %w", addr, err)
		}

		if !selectedAddr.IsValid() {
			selectedAddr = addr
			if selectedAddr.Is4() {
				break
			}
		} else if addr.Is4() || addr.Is4In6() {
			selectedAddr = addr
		}
	}

	if !selectedAddr.IsValid() {
		return fmt.Errorf("could not select any resolved address for %s in %s", host, strings.Join(ips, ", "))
	}

	cfg.GraylogAddress = net.JoinHostPort(selectedAddr.String(), port)

	return nil
}
