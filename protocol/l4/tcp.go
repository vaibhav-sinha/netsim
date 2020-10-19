package l4

import (
	"encoding/binary"
	"log"
	"netsim/protocol"
	"netsim/utils"
	"strconv"
	"sync"
	"time"
)

/*
TCP is a L4 protocol that is connection-oriented. Because of the connection oriented nature, it can provide reliable
communication and some other features like flow control, congestion control, etc. Here we will do a very simple implementation
with just reliable communication. That means that data will always arrive and in order. The implementation here is a very
inefficient way of ensuring reliability called stop-and-wait. Actual protocol uses a sliding window protocol.
TCP offers a byte based read and write interface.

Packet Format:

SrcPort 		- 2 bytes
DestPort		- 2 bytes
SequenceNum		- 4 bytes
Acknowledgement	- 4 bytes
Flags			- 1 byte
Checksum		- 1 byte
Data			- No fixed length

Flags:
SYN		- bit 1
FIN		- bit 2
ACK		- bit 3
RESET	- bit 4
*/
type TCP struct {
	identifier   []byte
	l3Protocols  []protocol.L3Protocol
	portBindings map[uint16]*TcpBinding
	lock         sync.Mutex
}

func NewTCP() *TCP {
	return &TCP{
		identifier:   protocol.TCP,
		portBindings: map[uint16]*TcpBinding{},
	}
}

/*
Next 3 methods make this a Protocol
*/
func (t *TCP) GetIdentifier() []byte {
	return t.identifier
}

func (t *TCP) SendUp(data []byte, metadata []byte, sender protocol.Protocol) {
	if !t.isValid(data) {
		log.Printf("UDP: Got corrupted packet")
		return
	}

	//Extract relevant information
	destPort := binary.BigEndian.Uint16(data[2:4])
	destAddr := metadata[4:8]

	b, found := t.portBindings[destPort]
	if !found {
		log.Printf("UDP: Got packet for port no one is listening on. Dropping.")
		return
	}

	if b.isMatch(destAddr, destPort) {
		b.sendUp(data, metadata, sender)
	} else {
		log.Printf("UDP: Got packet for different address. Dropping.")
	}
}

func (t *TCP) SendDown(data []byte, destAddr []byte, metadata []byte, sender protocol.Protocol) {
	//Not used. Data is sent using the connection.
}

/*
Following methods make this an implementation of L4 Protocol
*/
func (t *TCP) AddL3Protocol(l3Protocol protocol.L3Protocol) {
	t.l3Protocols = append(t.l3Protocols, l3Protocol)
}

/*
TCP public API
*/
func (t *TCP) Bind(ipAddr []byte, port uint16, networkProtocolIdentifier []byte) *TcpBinding {
	if t.IsPortInUse(port) {
		log.Printf("Error: Port already in use")
		return nil
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	b := newTcpBinding(t, ipAddr, port, networkProtocolIdentifier)
	t.portBindings[port] = b

	return b
}

func (t *TCP) IsPortInUse(port uint16) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	_, found := t.portBindings[port]
	return found
}

/*
Internal methods
*/
func (t *TCP) isValid(packet []byte) bool {
	actual := packet[13]
	calculated := utils.CalculateChecksum(packet)[0] - actual
	return actual == calculated
}

func (t *TCP) cleanup(b *TcpBinding) {
	delete(t.portBindings, b.port)
}

/*
Structure to track bindings
*/
type TcpBinding struct {
	tcp                       *TCP
	addr                      []byte
	port                      uint16
	networkProtocolIdentifier []byte
	listening                 bool
	backlogBuf                *utils.Buffer
	connections               map[string]*TcpConnection
}

func newTcpBinding(t *TCP, addr []byte, portNum uint16, networkProtocolIdentifier []byte) *TcpBinding {
	return &TcpBinding{
		tcp:                       t,
		addr:                      addr,
		port:                      portNum,
		networkProtocolIdentifier: networkProtocolIdentifier,
		connections:               map[string]*TcpConnection{},
	}
}

func (b *TcpBinding) Listen(backlog int) {
	b.listening = true
	b.backlogBuf = utils.NewBuffer(backlog + 1)
}

func (b *TcpBinding) Accept() *TcpConnection {
	//Check if Listen is set
	if !b.listening {
		log.Printf("TCP: Trying to Accept without Listening")
		return nil
	}

	//Pull out a connection request and create a connection object
	connectionRequest := b.backlogBuf.Get(true)
	connection := &TcpConnection{
		binding:         b,
		connectionDone:  make(chan bool),
		isSrc:           false,
		srcAddr:         connectionRequest[0:4],
		srcPort:         binary.BigEndian.Uint16(connectionRequest[4:6]),
		destAddr:        connectionRequest[6:10],
		destPort:        binary.BigEndian.Uint16(connectionRequest[10:12]),
		connectionState: 0,
		readBuffer:      utils.NewByteBuffer(defaultByteBufferSize),
		writeBuffer:     utils.NewByteBuffer(defaultByteBufferSize),
	}
	b.connections[b.getConnectionKey(connection.srcAddr, connection.srcPort)] = connection

	//Reply to the handshake request
	connection.ackConnectionRequest()

	//Wait until the connection is done
	<-connection.connectionDone
	return connection
}

func (b *TcpBinding) Connect(addr []byte, port uint16) *TcpConnection {
	//Check if Listen is set
	if b.listening {
		log.Printf("TCP: Trying to Connect while Listening")
		return nil
	}

	//Create a connection object
	connection := &TcpConnection{
		binding:         b,
		connectionDone:  make(chan bool),
		isSrc:           true,
		srcAddr:         b.addr,
		srcPort:         b.port,
		destAddr:        addr,
		destPort:        port,
		connectionState: 0,
		readBuffer:      utils.NewByteBuffer(defaultByteBufferSize),
		writeBuffer:     utils.NewByteBuffer(defaultByteBufferSize),
	}
	b.connections[b.getConnectionKey(addr, port)] = connection

	//Initiate to the handshake
	connection.triggerConnectionRequest()

	//Wait until the connection is done
	<-connection.connectionDone
	return connection
}

/*
Internal methods
*/
func (b *TcpBinding) sendUp(data []byte, metadata []byte, sender protocol.Protocol) {
	connectionKey := b.getConnectionKey(metadata[0:4], binary.BigEndian.Uint16(data[0:2]))
	connection, found := b.connections[connectionKey]
	if found {
		connection.sendUp(data, metadata, sender)
	} else {
		flag := data[12]
		if int(flag) == 2 {
			if b.listening {
				//Received SYN. Create connection requests
				var connectionRequest []byte
				connectionRequest = append(connectionRequest, metadata[0:4]...)
				connectionRequest = append(connectionRequest, data[0:2]...)
				connectionRequest = append(connectionRequest, metadata[4:8]...)
				connectionRequest = append(connectionRequest, data[2:4]...)

				//Queue the request
				b.backlogBuf.Put(connectionRequest)
				log.Printf("TCP: Got SYN")
			} else {
				log.Printf("TCP: Trying to connect on a port not in listening mode. Dropping.")
			}
		} else {
			log.Printf("TCP: Got invalid handshake request. Dropping.")
		}
	}
}

func (b *TcpBinding) getConnectionKey(addr []byte, port uint16) string {
	return string(addr) + strconv.Itoa(int(port))
}

func (b *TcpBinding) isMatch(destIp []byte, port uint16) bool {
	if port != b.port {
		return false
	}

	if binary.BigEndian.Uint32(b.addr) == 0 {
		return true
	}

	match := true
	for i := 0; i < 4; i++ {
		if destIp[i] != b.addr[i] {
			match = false
			break
		}
	}

	return match
}

func (b *TcpBinding) cleanup(t *TcpConnection) {
	key := b.getConnectionKey(t.srcAddr, t.srcPort)
	delete(b.connections, key)
	if len(b.connections) == 0 {
		b.tcp.cleanup(b)
	}
}

/*
Struct representing one connection
Data State: 0 -> Idle, 1 -> Waiting for ACK
Connection State: 0 -> Closed, 1 -> SYN Sent, 2 -> SYN Received, 3 -> Connected, 4 -> Closing, 5 -> Closed

Flags:
SYN		- bit 1
FIN		- bit 2
ACK		- bit 3
RESET	- bit 4
*/
type TcpConnection struct {
	binding         *TcpBinding
	connectionDone  chan bool
	isSrc           bool
	srcAddr         []byte
	destAddr        []byte
	srcPort         uint16
	destPort        uint16
	sendSeqNum      uint32
	recvSeqNum      uint32
	connectionState int
	dataState       int
	lastPacketSent  []byte
	readBuffer      *utils.ByteBuffer
	writeBuffer     *utils.ByteBuffer
}

/*
Public API
*/
func (t *TcpConnection) Send(b byte) {
	t.writeBuffer.Put(b)
}

func (t *TcpConnection) Recv() *byte {
	return t.readBuffer.Get(false)
}

func (t *TcpConnection) Close() {
	t.triggerTeardown()
}

/*
Internal methods
*/
func (t *TcpConnection) triggerConnectionRequest() {
	t.sendDown([]byte(""), byte(2))
	t.connectionState = 1
	log.Printf("TCP: SYN sent")
}

func (t *TcpConnection) ackConnectionRequest() {
	t.sendDown([]byte(""), byte(9))
	t.connectionState = 2
	log.Printf("TCP: SYN+ACK sent")
}

func (t *TcpConnection) completeHandshake() {
	t.sendDown([]byte(""), byte(8))
	t.connectionState = 3
	t.connectionDone <- true
	go t.sendPeriodically()
	log.Printf("TCP: ACK sent")
}

func (t *TcpConnection) triggerTeardown() {
	t.sendDown([]byte(""), byte(4))
	t.connectionState = 4
	log.Printf("TCP: FIN sent")
}

func (t *TcpConnection) ackTeardown() {
	t.sendDown([]byte(""), byte(12))
	t.connectionState = 4
	log.Printf("TCP: FIN+ACK sent")
}

func (t *TcpConnection) completeTeardown() {
	t.sendDown([]byte(""), byte(8))
	t.connectionState = 5
	t.binding.cleanup(t)
	log.Printf("TCP: ACK sent")
}

func (t *TcpConnection) triggerAckForPacket(data []byte) {
	t.sendDown([]byte(""), byte(8))
	t.recvSeqNum += 1
}

func (t *TcpConnection) sendPeriodically() {
	for t.connectionState == 3 {
		if t.dataState == 0 {
			var data []byte
			for {
				b := t.writeBuffer.Get(false)
				if b == nil {
					break
				}
				data = append(data, *b)
			}

			if len(data) > 0 {
				t.sendDown(data, byte(0))
				//go t.triggerRedeliveryOnTimer(data, t.sendSeqNum, time.NewTimer(2000 * time.Millisecond))
				t.sendSeqNum += 1
				t.dataState = 1
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (t *TcpConnection) triggerRedeliveryOnTimer(data []byte, seqNum uint32, timer *time.Timer) {
	<-timer.C
	if t.dataState == 1 && seqNum == t.sendSeqNum-1 {
		t.sendSeqNum -= 1
		t.sendDown(data, byte(0))
		t.sendSeqNum += 1
	}
}

func (t *TcpConnection) sendUp(data []byte, metadata []byte, sender protocol.Protocol) {
	flags := data[12]
	switch flags {
	case 0:
		for _, b := range data[14:] {
			t.readBuffer.Put(b)
		}
		t.triggerAckForPacket(data)
	case 4:
		if t.connectionState == 3 {
			t.ackTeardown()
		} else {
			log.Printf("TCP: Got unexpected FIN")
		}
	case 8:
		if t.connectionState == 2 {
			//Got ACK for final leg of 3-way handshake
			t.connectionState = 3
			t.connectionDone <- true
			go t.sendPeriodically()
		} else if t.connectionState == 4 {
			//Got ACK for final leg of teardown
			t.connectionState = 5
			t.binding.cleanup(t)
		} else if t.connectionState == 3 {
			// Got ACK for previously sent packet
			if binary.BigEndian.Uint32(t.lastPacketSent[4:8]) == binary.BigEndian.Uint32(data[8:12]) {
				t.dataState = 0
			} else {
				log.Printf("TCP: Got ACK for incorrect packet")
			}
		}
	case 9:
		if t.connectionState == 1 {
			t.completeHandshake()
		} else {
			log.Printf("TCP: Got unexpected SYN+ACK")
		}
	case 12:
		if t.connectionState == 4 {
			t.completeTeardown()
		} else {
			log.Printf("TCP: Got unexpected FIN+ACK")
		}
	}
}

func (t *TcpConnection) sendDown(data []byte, flags byte) {
	//Find which network protocol to use
	var l3Protocol protocol.L3Protocol
	for _, l3P := range t.binding.tcp.l3Protocols {
		l3PIdentifier := l3P.GetIdentifier()
		if t.binding.networkProtocolIdentifier[0] == l3PIdentifier[0] && t.binding.networkProtocolIdentifier[1] == l3PIdentifier[1] {
			l3Protocol = l3P
			break
		}
	}

	if l3Protocol == nil {
		log.Printf("Error: Could not find matching network protocol")
		return
	}

	srcPort := make([]byte, 2)
	destPort := make([]byte, 2)
	var destAddr []byte
	seqNum := make([]byte, 4)
	ackNum := make([]byte, 4)

	if t.isSrc {
		binary.BigEndian.PutUint16(srcPort, t.srcPort)
		binary.BigEndian.PutUint16(destPort, t.destPort)
		destAddr = t.destAddr
	} else {
		binary.BigEndian.PutUint16(srcPort, t.destPort)
		binary.BigEndian.PutUint16(destPort, t.srcPort)
		destAddr = t.srcAddr
	}

	//Normal packet
	if int(flags) == 0 {
		binary.BigEndian.PutUint32(seqNum, t.sendSeqNum)
	}

	//ACK packet
	if int(flags) == 8 {
		binary.BigEndian.PutUint32(ackNum, t.recvSeqNum)
	}

	//Create the packet
	var packet []byte
	packet = append(packet, srcPort...)
	packet = append(packet, destPort...)
	packet = append(packet, seqNum...)
	packet = append(packet, ackNum...)
	packet = append(packet, flags)
	packet = append(packet, byte(0))
	packet = append(packet, data...)

	//Fill in the checksum
	packet[13] = utils.CalculateChecksum(packet)[0]

	//Send the packet
	l3Protocol.SendDown(packet, destAddr, []byte{defaultTOS, defaultTTL}, t.binding.tcp)

	//Hold a copy for ACK checks
	t.lastPacketSent = packet
}
