package generator

import "net"

type NetworkInfo struct {
	ID      string
	Name    string
	Subnets []net.IPNet
}

func (n *NetworkInfo) MatchesID(id string) bool {
	return n.ID != "" && n.ID == id
}

func (n *NetworkInfo) MatchesName(name string) bool {
	return n.Name != "" && n.Name == name
}

func (n *NetworkInfo) ContainsIP(ip net.IP) bool {
	for _, subnet := range n.Subnets {
		if subnet.Contains(ip) {
			return true
		}
	}
	return false
}
