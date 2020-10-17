package l4

import (
	"encoding/binary"
	"log"
	"netsim/protocol"
	"netsim/utils"
	"sync"
)

/*
We will implement UDP which is a simple de-multiplexing protocol for process-to-process communication. I will keep the
implementation as simple as possible. That means no two processes can listen on same port no matter what. This is very
different than what happens in real OS implementations, if host has multiple IP addresses or SO_REUSEADDR is set. Also,
since UDP only needs to run on hosts, I am making the assumption that there is just one network interface on the host
and hence do not need to deal multi-host scenario.

Packet Format

SrcPort		- 2 byte
DestPort	- 2 byte
Length		- 2 byte
Checksum	- 1 byte
Data		- No fixed length
*/
type UDP struct {
	identifier   []byte
	l3Protocols  []protocol.L3Protocol
	portBindings map[uint16]*UdpBinding
	lock         sync.Mutex
}

func NewUDP() *UDP {
	return &UDP{
		identifier:   protocol.UDP,
		portBindings: map[uint16]*UdpBinding{},
	}
}

/*
Next 3 methods make this a Protocol
*/
func (u *UDP) GetIdentifier() []byte {
	return u.identifier
}

func (u *UDP) SendUp(data []byte, metadata []byte, sender protocol.Protocol) {
	if !u.isValid(data) {
		log.Printf("UDP: Got corrupted packet")
		return
	}

	//Extract relevant information
	destPort := binary.BigEndian.Uint16(data[2:4])
	destAddr := metadata[4:8]

	b, found := u.portBindings[destPort]
	if !found {
		log.Printf("UDP: Got packet for port no one is listening on. Dropping.")
		return
	}

	if b.isMatch(destAddr, destPort) {
		b.putInBuffer(data[7:])
	} else {
		log.Printf("UDP: Got packet for different address. Dropping.")
	}
}

func (u *UDP) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {
	//Extract relevant information
	destPort := metadata[0:2]
	srcPort := metadata[2:4]

	//Calculate packet length
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(7+len(data)))

	//Find which network protocol to use
	networkProtocolIdentifier := metadata[4:6]
	var l3Protocol protocol.L3Protocol
	for _, l3P := range u.l3Protocols {
		l3PIdentifier := l3P.GetIdentifier()
		if networkProtocolIdentifier[0] == l3PIdentifier[0] && networkProtocolIdentifier[1] == l3PIdentifier[1] {
			l3Protocol = l3P
			break
		}
	}

	if l3Protocol == nil {
		log.Printf("Error: Could not find matching network protocol")
		return
	}

	//Create the packet
	var packet []byte
	packet = append(packet, srcPort...)
	packet = append(packet, destPort...)
	packet = append(packet, length...)
	packet = append(packet, byte(0))
	packet = append(packet, data...)

	//Fill in the checksum
	packet[6] = utils.CalculateChecksum(packet)[0]

	//Send the packet
	l3Protocol.SendDown(packet, destAddr, []byte{defaultTOS, defaultTTL}, u)
}

/*
Following methods make this an implementation of L4 Protocol
*/
func (u *UDP) AddL3Protocol(l3Protocol protocol.L3Protocol) {
	u.l3Protocols = append(u.l3Protocols, l3Protocol)
}

/*
UDP public API
*/
func (u *UDP) Bind(ipAddr []byte, port uint16) *UdpBinding {
	if u.IsPortInUse(port) {
		log.Printf("Error: Port already in use")
		return nil
	}

	u.lock.Lock()
	defer u.lock.Unlock()

	b := newUdpBinding(u, ipAddr, port)
	u.portBindings[port] = b

	return b
}

func (u *UDP) IsPortInUse(port uint16) bool {
	u.lock.Lock()
	defer u.lock.Unlock()

	_, found := u.portBindings[port]
	return found
}

/*
Internal methods
*/
func (u *UDP) isValid(packet []byte) bool {
	actual := packet[6]
	calculated := utils.CalculateChecksum(packet)[0] - actual
	return actual == calculated
}

/*
Struct to track bindings
*/
type UdpBinding struct {
	udp    *UDP
	ip     []byte
	port   uint16
	buffer *utils.Buffer
}

func newUdpBinding(udp *UDP, ipAddr []byte, port uint16) *UdpBinding {
	return &UdpBinding{
		udp:    udp,
		ip:     ipAddr,
		port:   port,
		buffer: utils.NewBuffer(defaultBufferSize),
	}
}

func (b *UdpBinding) Close() {
	b.udp.lock.Lock()
	defer b.udp.lock.Unlock()

	delete(b.udp.portBindings, b.port)
}

func (b *UdpBinding) Recv() []byte {
	return b.buffer.Get(false)
}

func (b *UdpBinding) putInBuffer(item []byte) {
	b.buffer.Put(item)
}

func (b *UdpBinding) isMatch(destIp []byte, port uint16) bool {
	if port != b.port {
		return false
	}

	if binary.BigEndian.Uint32(b.ip) == 0 {
		return true
	}

	match := true
	for i := 0; i < 4; i++ {
		if destIp[i] != b.ip[i] {
			match = false
			break
		}
	}

	return match
}
