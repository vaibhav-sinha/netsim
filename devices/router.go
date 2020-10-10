package devices

import (
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
)

/*
A router is a device which connects multiple networks together
Based on the destination IP address in the packet, it forwards packet out of one of the interfaces after consulting the routing table.
A router generally implements routing algorithms to learn the routing table. In this implementation, the RouteProvider implements any routing algorithms.
We will be providing a StaticRouteProvider which is configured by a network administrator. If we want to implement a routing protocol then we can pass a RouteProvider as l4Protocols so it gets the packets and hence learn the routes.
*/
type Router struct {
	ip       *l3.IP
	numPorts int
}

func NewRouter(macs [][]byte, ipAddrs [][]byte, routingTable protocol.RouteProvider, addrResolutionTable protocol.AddressResolver) *Router {
	router := &Router{
		ip:       l3.NewIP(ipAddrs, true, nil, routingTable, addrResolutionTable),
		numPorts: len(ipAddrs),
	}

	for i, m := range macs {
		eth := l2.NewEthernet(hardware.NewEthernetAdapter(m, false), nil)
		eth.AddL3Protocol(router.ip)
		router.ip.SetL2ProtocolForInterface(i, eth)
	}

	return router
}

func (r *Router) GetL3Protocol() protocol.L3Protocol {
	return r.ip
}

func (r *Router) TurnOn() {
	for i := 0; i < r.numPorts; i++ {
		r.ip.GetL2ProtocolForInterface(i).GetAdapter().TurnOn()
	}
}

func (r *Router) TurnOff() {
	for i := 0; i < r.numPorts; i++ {
		r.ip.GetL2ProtocolForInterface(i).GetAdapter().TurnOn()
	}
}
