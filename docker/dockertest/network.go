package dockertest

import "shanhu.io/std/docker"

// Network is the fake daemon's internal model of a network.
type Network struct {
	ID     string
	Name   string
	Driver string

	// IPAMConfig, when non-empty, is exposed under IPAM.Config in inspect
	// responses.
	IPAMConfig []NetworkIPAMConfig
}

// NetworkIPAMConfig is one IPAM configuration entry on a Network.
type NetworkIPAMConfig struct {
	Subnet  string
	Gateway string
}

func (n *Network) toInfo() *docker.NetworkInfo {
	info := &docker.NetworkInfo{
		Name:   n.Name,
		ID:     n.ID,
		Driver: n.Driver,
	}
	if len(n.IPAMConfig) > 0 {
		cfg := make([]*docker.NetworkIPAMConfig, len(n.IPAMConfig))
		for i, c := range n.IPAMConfig {
			cfg[i] = &docker.NetworkIPAMConfig{
				Subnet:  c.Subnet,
				Gateway: c.Gateway,
			}
		}
		info.IPAM = &docker.NetworkIPAM{Config: cfg}
	}
	return info
}
