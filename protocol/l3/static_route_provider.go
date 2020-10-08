package l3

import "netsim/protocol"

/*
Static Routing Table
*/
type StaticRouteProvider struct {
	routingTable []*routingTableEntry
}

func NewStaticRouteProvider() *StaticRouteProvider {
	return &StaticRouteProvider{}
}

func (s *StaticRouteProvider) Add(cidr *protocol.CIDR, gateway []byte, intf int) {
	entry := &routingTableEntry{
		cidr:          cidr,
		gatewayIpAddr: gateway,
		intf:          intf,
	}
	s.routingTable = append(s.routingTable, entry)
}

func (s *StaticRouteProvider) GetGatewayForAddress(ipAddr []byte) []byte {
	return s.findMatchingEntry(ipAddr).gatewayIpAddr
}

func (s *StaticRouteProvider) GetInterfaceForAddress(ipAddr []byte) int {
	return s.findMatchingEntry(ipAddr).intf
}

func (s *StaticRouteProvider) findMatchingEntry(ipAddr []byte) *routingTableEntry {
	//Supporting only 8, 16, 24, 32 bit masks here for simplicity
	for _, entry := range s.routingTable {
		comparisonLength := entry.cidr.Mask / 8
		match := true
		for i := 0; i < comparisonLength; i++ {
			if entry.cidr.Address[i] != ipAddr[i] {
				match = false
				break
			}
		}
		if match {
			return entry
		}
	}

	//This should never happen if default gateway is added
	return nil
}

//Internal struct
type routingTableEntry struct {
	cidr          *protocol.CIDR
	gatewayIpAddr []byte
	intf          int
}
