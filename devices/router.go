package devices

import (
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
	"netsim/utils"
)

/*
A router is a device which connects multiple networks together
Based on the destination IP address in the packet, it forwards packet out of one of the interfaces after consulting the routing table.
A router generally implements routing algorithms to learn the routing table. In this implementation, the RouteProvider implements any routing algorithms.
We will be providing a StaticRouteProvider which is configured by a network administrator. If we want to implement a routing protocol then we can pass a RouteProvider as l4Protocols so it gets the packets and hence learn the routes.
*/
type Router struct {
	ports               []*l3.IP
	portMapping         map[protocol.FrameConsumer]int
	routingTable        protocol.RouteProvider
	addrResolutionTable protocol.AddressResolver
}

func NewRouter(macs [][]byte, ipAddrs [][]byte, routingTable protocol.RouteProvider, addrResolutionTable protocol.AddressResolver) *Router {
	router := &Router{
		portMapping:         make(map[protocol.FrameConsumer]int),
		routingTable:        routingTable,
		addrResolutionTable: addrResolutionTable,
	}

	for i, ipAddr := range ipAddrs {
		ip := l3.NewIP(ipAddr, true, nil, router, routingTable, addrResolutionTable)
		router.ports = append(router.ports, ip)
		router.portMapping[ip] = i
	}

	for i, m := range macs {
		l2.NewEthernet(hardware.NewEthernetAdapter(m, false), []protocol.L3Protocol{router.ports[i]}, nil)
	}

	return router
}

func (r *Router) TurnOn() {
	for _, a := range r.ports {
		a.GetL2Protocol().GetAdapter().TurnOn()
	}
}

func (r *Router) TurnOff() {
	for _, a := range r.ports {
		a.GetL2Protocol().GetAdapter().TurnOff()
	}
}

func (r *Router) GetPort(portNum int) *l3.IP {
	return r.ports[portNum]
}

func (r *Router) SendUp(packet []byte, metadata []byte, sender protocol.Protocol) {
	//Copy the packet
	newPacket := make([]byte, len(packet))
	copy(newPacket, packet)

	//Reduce the TTL for the packet
	ttl := int(newPacket[9])
	ttl -= 1

	//If TTL reached 0, then drop the packet
	if ttl == 0 {
		return
	}

	//Set the new TTL in packet
	newPacket[9] = byte(ttl)

	//Calculate the new checksum since packet has changed
	newPacket[11] = byte(0)
	newPacket[11] = utils.CalculateChecksum(newPacket[:20])[0]

	//Get the interface through which the packet has to leave
	destinationAddr := newPacket[16:20]
	intf := r.routingTable.GetInterfaceForAddress(destinationAddr)

	//If incoming interface is same as outgoing interface, then drop the packet
	if intf == r.portMapping[sender] {
		return
	}

	//Get the next hop address
	nextHopAddr := r.routingTable.GetGatewayForAddress(destinationAddr)
	l2Address := r.addrResolutionTable.Resolve(nextHopAddr)

	//Forward the packet
	r.ports[intf].GetL2Protocol().SendDown(newPacket, l2Address, nil, sender)
}
