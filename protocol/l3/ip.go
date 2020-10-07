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
	ipIdentMap          map[string]uint16
	buffer              map[uint64]*fragmentTracker
	ipAddress           []byte
	forwardingMode      bool
	version             []byte
	identifier          []byte
	l2Protocol          protocol.L2Protocol
	l4Protocols         []protocol.L4Protocol
	rawConsumer         protocol.FrameConsumer
	routingTable        protocol.RouteProvider
	addrResolutionTable protocol.AddressResolver
	lock                sync.Mutex
}

/*
Constructor
*/
func NewIP(ipAddress []byte, forwardingMode bool, l4Protocols []protocol.L4Protocol, rawConsumer protocol.FrameConsumer, routingTable protocol.RouteProvider, addrResolutionTable protocol.AddressResolver) *IP {
	ip := &IP{
		ipIdentMap:          map[string]uint16{},
		buffer:              map[uint64]*fragmentTracker{},
		ipAddress:           ipAddress,
		forwardingMode:      forwardingMode,
		version:             utils.HexStringToBytes("04"),
		identifier:          utils.HexStringToBytes("0800"),
		l4Protocols:         l4Protocols,
		rawConsumer:         rawConsumer,
		routingTable:        routingTable,
		addrResolutionTable: addrResolutionTable,
	}

	for _, l4 := range l4Protocols {
		l4.SetL3Protocol(ip)
	}

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
	tos := metadata[0]
	ttl := metadata[1]
	proto := l4Protocol.GetIdentifier()
	nextHopAddr := ip.routingTable.GetGatewayForAddress(destAddr)
	l2Address := ip.addrResolutionTable.Resolve(nextHopAddr)

	//Fragmentation logic follows. We use ident=0 for packets in which no fragmentation occurs
	if len(data) <= ip.l2Protocol.GetMTU() {
		packet := ip.createPacket(data, destAddr, tos, []byte{0, 0}, byte(1), []byte{0, 0}, ttl, proto)
		ip.l2Protocol.SendDown(packet, l2Address, nil, ip)
	} else {
		//Get the identifier to use
		ip.lock.Lock()
		var identifier uint16
		usedIdent, ok := ip.ipIdentMap[string(destAddr)]
		if ok {
			identifier = usedIdent + 1
		} else {
			identifier = 1
		}
		ip.ipIdentMap[string(destAddr)] = identifier
		ip.lock.Unlock()

		//Create fragments and send
		for totalBytesConsumed := 0; totalBytesConsumed < len(data); totalBytesConsumed += ip.l2Protocol.GetMTU() - 20 {
			//Flag indicated if this is the last fragment
			flag := 0
			if totalBytesConsumed+ip.l2Protocol.GetMTU()-20 >= len(data) {
				flag = 1
			}

			//Offset of bytes in this fragment
			var offset = make([]byte, 2)
			binary.BigEndian.PutUint16(offset, uint16(totalBytesConsumed))

			var ident = make([]byte, 2)
			binary.BigEndian.PutUint16(ident, identifier)

			//Create and send the packet
			endIndex := totalBytesConsumed + ip.l2Protocol.GetMTU() - 20
			if endIndex > len(data) {
				endIndex = len(data)
			}
			packet := ip.createPacket(data[totalBytesConsumed:endIndex], destAddr, tos, ident, byte(flag), offset, ttl, proto)
			ip.l2Protocol.SendDown(packet, l2Address, nil, ip)
		}
	}
}

func (ip *IP) SendUp(packet []byte, source protocol.FrameConsumer) {
	isValid := ip.isValidPacket(packet)
	if isValid {
		isPacketForMe := ip.isPacketForMe(packet)
		if !ip.forwardingMode && !isPacketForMe {
			return
		}

		ready, data := ip.reassemble(packet)
		if ready {
			if ip.rawConsumer != nil {
				ip.rawConsumer.SendUp(data, ip)
			}

			if len(ip.l4Protocols) > 0 {
				proto := packet[10]
				var upperLayerProtocol protocol.L4Protocol
				for _, l4P := range ip.l4Protocols {
					if l4P.GetIdentifier()[0] == proto {
						upperLayerProtocol = l4P
						break
					}
				}

				if !isPacketForMe {
					return
				}

				if upperLayerProtocol != nil {
					upperLayerProtocol.SendUp(data, ip)
				} else {
					log.Printf("IP: addr %s: Got unrecognized packet type: %v", string(ip.ipAddress), proto)
				}
			}
		}
	} else {
		log.Printf("IP: Got corrupted packet")
	}
}

/*
Next 2 methods make this an implementation of L3Protocol
*/
func (ip *IP) SetL2Protocol(l2Protocol protocol.L2Protocol) {
	ip.l2Protocol = l2Protocol
}

func (ip *IP) GetL2Protocol() protocol.L2Protocol {
	return ip.l2Protocol
}

/*
Internal methods
*/
func (ip *IP) createPacket(data []byte, destAddr []byte, tos byte, ident []byte, flags byte, offset []byte, ttl byte, proto []byte) []byte {
	var packetLength = make([]byte, 2)
	binary.BigEndian.PutUint16(packetLength, uint16(len(data)+20))

	b := []byte{}
	b = append(b, ip.version...)
	b = append(b, tos)
	b = append(b, packetLength...)
	b = append(b, ident...)
	b = append(b, flags)
	b = append(b, offset...)
	b = append(b, ttl)
	b = append(b, proto...)
	b = append(b, byte(0))
	b = append(b, ip.ipAddress...)
	b = append(b, destAddr...)
	b = append(b, data...)

	b[11] = utils.CalculateChecksum(b[:20])[0]
	return b
}

func (ip *IP) getBufferKey(sourceAddr []byte, ident []byte) uint64 {
	k := []byte{}
	k = append(k, []byte{0, 0}...)
	k = append(k, sourceAddr...)
	k = append(k, ident...)
	return binary.BigEndian.Uint64(k)
}

func (ip *IP) reassemble(packet []byte) (bool, []byte) {
	ip.lock.Lock()
	defer ip.lock.Unlock()

	//Extract relevant info from packet
	sourceAddr := packet[12:16]
	ident := packet[4:6]

	identifier := binary.BigEndian.Uint16(ident)

	//Return data if packet is un-fragmented
	if identifier == 0 {
		return true, packet[20:]
	}

	//Add packet to buffer
	key := ip.getBufferKey(sourceAddr, ident)
	tracker, ok := ip.buffer[key]
	if ok {
		tracker.lastFragmentAt = time.Now()
		tracker.packets = append(tracker.packets, packet)
	} else {
		tracker = &fragmentTracker{
			lastFragmentAt: time.Now(),
			packets:        [][]byte{packet},
		}
		ip.buffer[key] = tracker
	}

	//Check if all fragments have arrived
	ready, packets := ip.isReadyForReassembly(tracker)
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

func (ip *IP) isReadyForReassembly(tracker *fragmentTracker) (bool, [][]byte) {
	maxDataPerPacket := 65536 - 20
	maxDataPerFragment := ip.l2Protocol.GetMTU() - 20
	maxFragments := maxDataPerPacket / maxDataPerFragment

	var sortedPackets = make([][]byte, maxFragments)
	for _, packet := range tracker.packets {
		offset := binary.BigEndian.Uint16(packet[7:9])
		sortedPackets[int(offset)/maxDataPerFragment] = packet
	}

	isReady := true
	i := 0
	for ; i < maxFragments; i++ {
		if sortedPackets[i] == nil {
			isReady = false
			break
		}
		if int(sortedPackets[i][6]) == 1 {
			break
		}
	}

	if !isReady {
		return false, nil
	}

	return true, sortedPackets[:i+1]
}

func (ip *IP) isValidPacket(packet []byte) bool {
	var header = make([]byte, 20)
	copy(header, packet[0:20])
	header[11] = byte(0)

	calculated := utils.CalculateChecksum(header)[0]
	actual := packet[11]

	return calculated == actual
}

func (ip *IP) isPacketForMe(packet []byte) bool {
	destinationAddr := packet[16:20]

	for i := 0; i < 4; i++ {
		if ip.ipAddress[i] != destinationAddr[i] {
			return false
		}
	}

	return true
}

func (ip *IP) cleanBuffersPeriodically() {
	for {
		ip.lock.Lock()
		now := time.Now()

		var expiredKeys []uint64
		for key, tracker := range ip.buffer {
			if tracker.lastFragmentAt.Add(identifierExpiryDuration).After(now) {
				expiredKeys = append(expiredKeys, key)
			}
		}

		for _, k := range expiredKeys {
			delete(ip.buffer, k)
		}
		ip.lock.Unlock()
		time.Sleep(600 * time.Second)
	}
}

/*
Internal struct to track fragments
*/
type fragmentTracker struct {
	lastFragmentAt time.Time
	packets        [][]byte
}
