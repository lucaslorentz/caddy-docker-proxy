package generator

import "net"

type NetworkGroup struct {
	Name     string
	Networks []*NetworkInfo
}

func (n *NetworkGroup) MatchesID(id string) bool {
	for _, selector := range n.Networks {
		if selector.MatchesID(id) {
			return true
		}
	}
	return false
}

func (n *NetworkGroup) MatchesName(name string) bool {
	for _, selector := range n.Networks {
		if selector.MatchesName(name) {
			return true
		}
	}
	return false
}

func (n *NetworkGroup) ContainsIP(ip net.IP) bool {
	for _, selector := range n.Networks {
		if selector.ContainsIP(ip) {
			return true
		}
	}
	return false
}
