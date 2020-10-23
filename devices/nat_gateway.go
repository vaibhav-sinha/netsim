package devices

import (
	"encoding/binary"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/protocol/l2"
	"netsim/protocol/l3"
	"netsim/utils"
	"strconv"
	"strings"
	"sync"
)

/*
A NAT Gateway is a router which does network address translation so that devices which do not have a public IP address
can communicate with the outside world. In an actual implementation, the mappings would have a LRU eviction policy, but
no eviction is being implemented here since in simulations we are never going to run out of ports.
*/
type NatGateway struct {
	ip                  *l3.IP
	numPorts            int
	mapping             map[string]uint16
	revMapping          map[uint16]string
	routingTable        protocol.RouteProvider
	addrResolutionTable protocol.AddressResolver
	lock                sync.Mutex
}

func NewNatGateway(macs [][]byte, ipAddrs [][]byte, routingTable protocol.RouteProvider, addrResolutionTable protocol.AddressResolver) *NatGateway {
	router := &NatGateway{
		numPorts:            len(ipAddrs),
		mapping:             map[string]uint16{},
		revMapping:          map[uint16]string{},
		routingTable:        routingTable,
		addrResolutionTable: addrResolutionTable,
	}

	router.ip = l3.NewIP(ipAddrs, false, router, routingTable, addrResolutionTable)

	for i, m := range macs {
		eth := l2.NewEthernet(hardware.NewEthernetAdapter(m, false), nil)
		eth.AddL3Protocol(router.ip)
		router.ip.SetL2ProtocolForInterface(i, eth)
	}

	return router
}

func (r *NatGateway) GetL3Protocol() protocol.L3Protocol {
	return r.ip
}

func (r *NatGateway) TurnOn() {
	for i := 0; i < r.numPorts; i++ {
		r.ip.GetL2ProtocolForInterface(i).GetAdapter().TurnOn()
	}
}

func (r *NatGateway) TurnOff() {
	for i := 0; i < r.numPorts; i++ {
		r.ip.GetL2ProtocolForInterface(i).GetAdapter().TurnOn()
	}
}

func (r *NatGateway) SendUp(packet []byte, metadata []byte, source protocol.Protocol) {
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

	//Get the interface through which the packet has to leave
	destinationAddr := newPacket[16:20]
	intf := r.routingTable.GetInterfaceForAddress(destinationAddr)

	if r.requiresForwardTranslation(newPacket[12:16], newPacket[16:20]) {
		sourceIpAddr := newPacket[12:16]
		sourcePort := r.getSourcePort(newPacket)
		key := r.getKey(sourceIpAddr, sourcePort)

		r.lock.Lock()
		//Check if a mapping already exists, else create one
		mappedPort, ok := r.mapping[key]
		if !ok {
			//Get a free port
			mappedPort = r.getFreePort()

			//Create mapping
			r.mapping[key] = mappedPort
			r.revMapping[mappedPort] = key
		}

		//Change the source IP addr for the outgoing packet
		for i := 0; i < 4; i++ {
			newPacket[12+i] = r.ip.GetAddressForInterface(intf)[i]
		}

		//Change the source port for the outgoing packet
		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, mappedPort)
		for i := 0; i < 2; i++ {
			newPacket[20+i] = portBytes[i]
		}

		//Change the TCP/UDP checksum since the packet has changed
		if r.isTcpPacket(newPacket) {
			newPacket[20+13] = 0
			newPacket[20+13] = utils.CalculateChecksum(newPacket[20:])[0]
		} else {
			newPacket[20+6] = 0
			newPacket[20+6] = utils.CalculateChecksum(newPacket[20:])[0]
		}
		r.lock.Unlock()
	}

	if r.requiresReverseTranslation(newPacket[12:16], newPacket[16:20]) {
		destinationPort := r.getDestinationPort(newPacket)

		r.lock.Lock()
		//Check if a mapping exists, else don't do anything
		key, ok := r.revMapping[destinationPort]
		if ok {
			mappedIpAddr, mappedPort := r.getIpAndPort(key)
			destinationAddr = mappedIpAddr
			intf = r.routingTable.GetInterfaceForAddress(destinationAddr)

			//Change the destination IP addr for the outgoing packet
			for i := 0; i < 4; i++ {
				newPacket[16+i] = mappedIpAddr[i]
			}

			//Change the destination port for the outgoing packet
			portBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(portBytes, mappedPort)
			for i := 0; i < 2; i++ {
				newPacket[22+i] = portBytes[i]
			}

			//Change the TCP/UDP checksum since the packet has changed
			if r.isTcpPacket(newPacket) {
				newPacket[20+13] = 0
				newPacket[20+13] = utils.CalculateChecksum(newPacket[20:])[0]
			} else {
				newPacket[20+6] = 0
				newPacket[20+6] = utils.CalculateChecksum(newPacket[20:])[0]
			}
		}
		r.lock.Unlock()
	}

	//Calculate the new checksum since packet has changed
	newPacket[11] = byte(0)
	newPacket[11] = utils.CalculateChecksum(newPacket[:20])[0]

	//If incoming interface is same as outgoing interface, then drop the packet
	if intf == r.getInterfaceNum(source) {
		return
	}

	//Get the next hop address
	nextHopAddr := r.routingTable.GetGatewayForAddress(destinationAddr)
	l2Address := r.addrResolutionTable.Resolve(nextHopAddr)

	//Forward the packet
	r.ip.GetL2ProtocolForInterface(intf).SendDown(newPacket, l2Address, nil, r.ip)
}

func (r *NatGateway) requiresForwardTranslation(sourceIpAddr []byte, destinationIpAddr []byte) bool {
	return r.isPrivateIp(sourceIpAddr) && !r.isPrivateIp(destinationIpAddr)
}

func (r *NatGateway) requiresReverseTranslation(sourceIpAddr []byte, destinationIpAddr []byte) bool {
	return !r.isPrivateIp(sourceIpAddr)
}

func (r *NatGateway) isPrivateIp(ipAddr []byte) bool {
	//Class A check
	if int(ipAddr[0]) == 10 {
		return true
	}

	//Class B check
	if int(ipAddr[0]) == 172 && int(ipAddr[1]) >= 16 && int(ipAddr[1]) <= 31 {
		return true
	}

	//Class C check
	if int(ipAddr[0]) == 192 && int(ipAddr[1]) == 168 {
		return true
	}

	return false
}

func (r *NatGateway) getKey(ipAddr []byte, port uint16) string {
	return string(ipAddr) + ":" + strconv.Itoa(int(port))
}

func (r *NatGateway) getIpAndPort(key string) ([]byte, uint16) {
	parts := strings.Split(key, ":")
	ipAddr := []byte(parts[0])
	port, _ := strconv.Atoi(parts[1])

	return ipAddr, uint16(port)
}

func (r *NatGateway) getSourcePort(packet []byte) uint16 {
	port := packet[20:22]
	return binary.BigEndian.Uint16(port)
}

func (r *NatGateway) getDestinationPort(packet []byte) uint16 {
	port := packet[22:24]
	return binary.BigEndian.Uint16(port)
}

func (r *NatGateway) getInterfaceNum(source protocol.Protocol) int {
	for i := 0; i < r.numPorts; i++ {
		if source == r.ip.GetL2ProtocolForInterface(i) {
			return i
		}
	}

	return -1
}

func (r *NatGateway) getFreePort() uint16 {
	var i uint16
	for i = 0; i <= uint16(65535); i++ {
		_, ok := r.revMapping[i]
		if !ok {
			return i
		}
	}

	//If this happens then it means we ran out of ports. Not handling this, for simplicity.
	return 0
}

func (r *NatGateway) isTcpPacket(packet []byte) bool {
	ident := packet[10]
	return ident == protocol.TCP[0]
}
