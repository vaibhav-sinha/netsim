package l3

import (
	"encoding/binary"
	"log"
	"netsim/protocol"
	"netsim/utils"
	"sync"
	"time"
)

/*
We will implement a simple IP-like protocol. High level differences from IP are:
1. The header will be fixed length. No need of HLen field
2. There won't be any options. No padding will be needed
3. All fields will be at least 1 byte long

Packet Format:

Version 		- 1 byte
TOS     		- 1 byte
Length  		- 2 byte
Ident   		- 2 byte
Flags   		- 1 byte
Offset  		- 2 byte
TTL     		- 1 byte
Protocol 		- 1 byte
Checksum 		- 1 byte
SourceAddr      - 4 byte
DestinationAddr - 4 byte
Data			- No fixed length but should be less than 2^16 - 20 bytes
*/

const (
	identifierExpiryDuration = 10 * time.Second
)

type IP struct {
	forwardingMode      bool
	version             []byte
	identifier          []byte
	interfaces          []*ipInterface
	l4Protocols         []protocol.L4Protocol
	rawConsumer         protocol.FrameConsumer
	routingTable        protocol.RouteProvider
	addrResolutionTable protocol.AddressResolver
	lock                sync.Mutex
}

/*
Constructor
*/
func NewIP(ipAddresses [][]byte, forwardingMode bool, rawConsumer protocol.FrameConsumer, routingTable protocol.RouteProvider, addrResolutionTable protocol.AddressResolver) *IP {
	ip := &IP{
		forwardingMode:      forwardingMode,
		version:             utils.HexStringToBytes("04"),
		identifier:          protocol.IP,
		rawConsumer:         rawConsumer,
		routingTable:        routingTable,
		addrResolutionTable: addrResolutionTable,
	}

	var interfaces []*ipInterface
	for _, ipAddr := range ipAddresses {
		interfaces = append(interfaces, newIPInterface(ipAddr, ip))
	}
	ip.interfaces = interfaces

	go ip.cleanBuffersPeriodically()
	return ip
}

/*
Next 3 methods make this an implementation of Protocol
*/
func (ip *IP) GetIdentifier() []byte {
	return ip.identifier
}

func (ip *IP) SendDown(data []byte, destAddr []byte, metadata []byte, l4Protocol protocol.Protocol) {
	intfNum := ip.routingTable.GetInterfaceForAddress(destAddr)
	ip.interfaces[intfNum].sendDown(data, destAddr, metadata, l4Protocol)
}

func (ip *IP) SendUp(packet []byte, metadata []byte, source protocol.Protocol) {
	isValid := ip.isValidPacket(packet)
	if isValid {
		if ip.rawConsumer != nil {
			ip.rawConsumer.SendUp(packet, nil, ip)
		}

		isPacketForMe, intfNum := ip.isPacketForMe(packet)
		if isPacketForMe {
			ip.interfaces[intfNum].sendUp(packet, metadata, source)
		} else {
			if ip.forwardingMode {
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
				intf := ip.routingTable.GetInterfaceForAddress(destinationAddr)

				//If incoming interface is same as outgoing interface, then drop the packet
				if intf == ip.getInterfaceNum(source) {
					return
				}

				//Get the next hop address
				nextHopAddr := ip.routingTable.GetGatewayForAddress(destinationAddr)
				l2Address := ip.addrResolutionTable.Resolve(nextHopAddr)

				//Forward the packet
				ip.interfaces[intf].l2Protocol.SendDown(newPacket, l2Address, nil, ip)
			}
		}
	} else {
		log.Printf("IP: Got corrupted packet")
	}
}

/*
Next 4 methods make this an implementation of L3Protocol
*/
func (ip *IP) SetL2ProtocolForInterface(intfNum int, l2Protocol protocol.L2Protocol) {
	ip.interfaces[intfNum].l2Protocol = l2Protocol
}

func (ip *IP) GetL2ProtocolForInterface(intfNum int) protocol.L2Protocol {
	return ip.interfaces[intfNum].l2Protocol
}

func (ip *IP) GetAddressForInterface(intfNum int) []byte {
	return ip.interfaces[intfNum].ipAddress
}

func (ip *IP) AddL4Protocol(l4Protocol protocol.L4Protocol) {
	ip.l4Protocols = append(ip.l4Protocols, l4Protocol)
}

/*
Internal methods
*/
func (ip *IP) isValidPacket(packet []byte) bool {
	var header = make([]byte, 20)
	copy(header, packet[0:20])
	header[11] = byte(0)

	calculated := utils.CalculateChecksum(header)[0]
	actual := packet[11]

	return calculated == actual
}

func (ip *IP) isPacketForMe(packet []byte) (bool, int) {
	destinationAddr := packet[16:20]

	for i := 0; i < len(ip.interfaces); i++ {
		match := true
		for j := 0; j < 4; j++ {
			if ip.interfaces[i].ipAddress[j] != destinationAddr[j] {
				match = false
				break
			}
		}
		if match {
			return true, i
		}
	}

	return false, -1
}

func (ip *IP) getInterfaceNum(source protocol.Protocol) int {
	for i, intf := range ip.interfaces {
		if source == intf.l2Protocol {
			return i
		}
	}

	return -1
}

func (ip *IP) cleanBuffersPeriodically() {
	for {
		for _, intf := range ip.interfaces {
			intf.cleanBuffers()
		}
		time.Sleep(600 * time.Second)
	}
}

/*
Per-interface struct
*/
type ipInterface struct {
	ipIdentMap map[string]uint16
	buffer     map[uint64]*fragmentTracker
	ipAddress  []byte
	l2Protocol protocol.L2Protocol
	lock       sync.Mutex
	ip         *IP
}

func newIPInterface(ipAddress []byte, ip *IP) *ipInterface {
	return &ipInterface{
		ipIdentMap: make(map[string]uint16),
		buffer:     make(map[uint64]*fragmentTracker),
		ipAddress:  ipAddress,
		ip:         ip,
	}
}

func (i *ipInterface) getAddress() []byte {
	return i.ipAddress
}

func (i *ipInterface) setL2Protocol(l2Protocol protocol.L2Protocol) {
	i.l2Protocol = l2Protocol
}

func (i *ipInterface) getL2Protocol() protocol.L2Protocol {
	return i.l2Protocol
}

func (i *ipInterface) sendUp(packet []byte, metadata []byte, source protocol.Protocol) {
	ready, data := i.reassemble(packet)
	if ready {
		//Extract relevant info from packet
		sourceAddr := packet[12:16]
		destinationAddr := packet[16:20]

		ipMetadata := []byte{}
		ipMetadata = append(ipMetadata, sourceAddr...)
		ipMetadata = append(ipMetadata, destinationAddr...)

		if len(i.ip.l4Protocols) > 0 {
			proto := packet[10]
			var upperLayerProtocol protocol.L4Protocol
			for _, l4P := range i.ip.l4Protocols {
				if l4P.GetIdentifier()[0] == proto {
					upperLayerProtocol = l4P
					break
				}
			}

			if upperLayerProtocol != nil {
				upperLayerProtocol.SendUp(data, ipMetadata, i.ip)
			} else {
				log.Printf("IP: addr %s: Got unrecognized packet type: %v", string(i.ipAddress), proto)
			}
		}
	}
}

func (i *ipInterface) sendDown(data []byte, destAddr []byte, metadata []byte, l4Protocol protocol.Protocol) {
	tos := metadata[0]
	ttl := metadata[1]
	proto := l4Protocol.GetIdentifier()
	nextHopAddr := i.ip.routingTable.GetGatewayForAddress(destAddr)
	l2Address := i.ip.addrResolutionTable.Resolve(nextHopAddr)

	//Fragmentation logic follows. We use ident=0 for packets in which no fragmentation occurs
	if len(data) <= i.l2Protocol.GetMTU() {
		packet := i.createPacket(data, destAddr, tos, []byte{0, 0}, byte(1), []byte{0, 0}, ttl, proto)
		i.l2Protocol.SendDown(packet, l2Address, nil, i.ip)
	} else {
		//Get the identifier to use
		i.lock.Lock()
		var identifier uint16
		usedIdent, ok := i.ipIdentMap[string(destAddr)]
		if ok {
			identifier = usedIdent + 1
		} else {
			identifier = 1
		}
		i.ipIdentMap[string(destAddr)] = identifier
		i.lock.Unlock()

		//Create fragments and send
		for totalBytesConsumed := 0; totalBytesConsumed < len(data); totalBytesConsumed += i.l2Protocol.GetMTU() - 20 {
			//Flag indicates if this is the last fragment
			flag := 0
			if totalBytesConsumed+i.l2Protocol.GetMTU()-20 >= len(data) {
				flag = 1
			}

			//Offset of bytes in this fragment
			var offset = make([]byte, 2)
			binary.BigEndian.PutUint16(offset, uint16(totalBytesConsumed))

			var ident = make([]byte, 2)
			binary.BigEndian.PutUint16(ident, identifier)

			//Create and send the packet
			endIndex := totalBytesConsumed + i.l2Protocol.GetMTU() - 20
			if endIndex > len(data) {
				endIndex = len(data)
			}
			packet := i.createPacket(data[totalBytesConsumed:endIndex], destAddr, tos, ident, byte(flag), offset, ttl, proto)
			i.l2Protocol.SendDown(packet, l2Address, nil, i.ip)
		}
	}
}

func (i *ipInterface) cleanBuffers() {
	i.lock.Lock()
	now := time.Now()

	var expiredKeys []uint64
	for key, tracker := range i.buffer {
		if tracker.lastFragmentAt.Add(identifierExpiryDuration).After(now) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, k := range expiredKeys {
		delete(i.buffer, k)
	}
	i.lock.Unlock()
}

func (i *ipInterface) isReadyForReassembly(tracker *fragmentTracker) (bool, [][]byte) {
	maxDataPerPacket := 65536 - 20
	maxDataPerFragment := i.l2Protocol.GetMTU() - 20
	maxFragments := maxDataPerPacket / maxDataPerFragment

	var sortedPackets = make([][]byte, maxFragments)
	for _, packet := range tracker.packets {
		offset := binary.BigEndian.Uint16(packet[7:9])
		sortedPackets[int(offset)/maxDataPerFragment] = packet
	}

	isReady := true
	j := 0
	for ; j < maxFragments; j++ {
		if sortedPackets[j] == nil {
			isReady = false
			break
		}
		if int(sortedPackets[j][6]) == 1 {
			break
		}
	}

	if !isReady {
		return false, nil
	}

	return true, sortedPackets[:j+1]
}

func (i *ipInterface) reassemble(packet []byte) (bool, []byte) {
	i.lock.Lock()
	defer i.lock.Unlock()

	//Extract relevant info from packet
	sourceAddr := packet[12:16]
	ident := packet[4:6]

	identifier := binary.BigEndian.Uint16(ident)

	//Return data if packet is un-fragmented
	if identifier == 0 {
		return true, packet[20:]
	}

	//Add packet to buffer
	key := i.getBufferKey(sourceAddr, ident)
	tracker, ok := i.buffer[key]
	if ok {
		tracker.lastFragmentAt = time.Now()
		tracker.packets = append(tracker.packets, packet)
	} else {
		tracker = &fragmentTracker{
			lastFragmentAt: time.Now(),
			packets:        [][]byte{packet},
		}
		i.buffer[key] = tracker
	}

	//Check if all fragments have arrived
	ready, packets := i.isReadyForReassembly(tracker)
	if !ready {
		return false, nil
	}

	//Reassemble
	var data []byte
	for _, p := range packets {
		data = append(data, p[20:]...)
	}
	return true, data
}

func (i *ipInterface) getBufferKey(sourceAddr []byte, ident []byte) uint64 {
	k := []byte{}
	k = append(k, []byte{0, 0}...)
	k = append(k, sourceAddr...)
	k = append(k, ident...)
	return binary.BigEndian.Uint64(k)
}

func (i *ipInterface) createPacket(data []byte, destAddr []byte, tos byte, ident []byte, flags byte, offset []byte, ttl byte, proto []byte) []byte {
	var packetLength = make([]byte, 2)
	binary.BigEndian.PutUint16(packetLength, uint16(len(data)+20))

	b := []byte{}
	b = append(b, i.ip.version...)
	b = append(b, tos)
	b = append(b, packetLength...)
	b = append(b, ident...)
	b = append(b, flags)
	b = append(b, offset...)
	b = append(b, ttl)
	b = append(b, proto...)
	b = append(b, byte(0))
	b = append(b, i.ipAddress...)
	b = append(b, destAddr...)
	b = append(b, data...)

	b[11] = utils.CalculateChecksum(b[:20])[0]
	return b
}

/*
Internal struct to track fragments
*/
type fragmentTracker struct {
	lastFragmentAt time.Time
	packets        [][]byte
}
