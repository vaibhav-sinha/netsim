package api

import (
	"encoding/binary"
	"log"
	"math/rand"
	"netsim/protocol"
	"netsim/protocol/l4"
)

// L3 constants
var (
	AF_INET = 0
)

// L4 constants
var (
	SOCK_DGRAM  = 0
	SOCK_STREAM = 1
)

// Socket type
const (
	UDP = 0
	TCP = 1
)

/*
Socket is a high level API that abstracts the protocol being used and offers a uniform interface for applications to
talk to the network. Though it is a (very) leaky abstraction, but still very useful.
*/
type Socket struct {
	host          Host
	sockType      int
	networkType   int
	udpBinding    *l4.UdpBinding
	tcpBinding    *l4.TcpBinding
	tcpConnection *l4.TcpConnection
	data          []byte
}

func NewSocket(host Host, domain int, channelType int, protocol int) *Socket {
	s := &Socket{
		host:        host,
		networkType: domain,
	}

	if channelType == SOCK_DGRAM && protocol == 0 {
		s.sockType = UDP
	}

	if channelType == SOCK_STREAM && protocol == 0 {
		s.sockType = TCP
	}

	return s
}

func newClientSocket(listeningSocket *Socket, tcpConnection *l4.TcpConnection) *Socket {
	s := &Socket{
		host:          listeningSocket.host,
		sockType:      listeningSocket.sockType,
		networkType:   listeningSocket.networkType,
		tcpBinding:    listeningSocket.tcpBinding,
		tcpConnection: tcpConnection,
	}

	return s
}

/*
The Socket API
*/
func (s *Socket) Bind(ipAddr []byte, port uint16) {
	var networkProtocolIdentifier []byte
	if s.networkType == AF_INET {
		networkProtocolIdentifier = protocol.IP
	}

	if s.sockType == UDP {
		s.udpBinding = s.host.GetUDP().Bind(ipAddr, port, networkProtocolIdentifier)
	}
	if s.sockType == TCP {
		s.tcpBinding = s.host.GetTCP().Bind(ipAddr, port, networkProtocolIdentifier)
	}
}

func (s *Socket) Listen(backlog int) {
	if s.sockType == TCP {
		s.tcpBinding.Listen(backlog)
	}
}

func (s *Socket) Accept() *Socket {
	if s.sockType == TCP {
		conn := s.tcpBinding.Accept()
		socket := newClientSocket(s, conn)
		return socket
	}

	return nil
}

func (s *Socket) Connect(destAddr []byte, destPort uint16) {
	if s.sockType == TCP {
		if s.tcpBinding == nil {
			randomPort := s.getRandomPort()
			s.Bind([]byte{0, 0, 0, 0}, randomPort)
		}
		s.tcpConnection = s.tcpBinding.Connect(destAddr, destPort)
	}
}

func (s *Socket) SendTo(destAddr []byte, destPort uint16, srcPort *uint16, data []byte) {
	if s.sockType == UDP {
		metadata := make([]byte, 4)
		//Populate destPort
		binary.BigEndian.PutUint16(metadata[0:2], destPort)

		//Generate random srcPort if needed and populate
		if srcPort == nil {
			randomPort := s.getRandomPort()
			srcPort = &randomPort
		}
		binary.BigEndian.PutUint16(metadata[2:4], *srcPort)

		//Populate the network protocol
		if s.networkType == AF_INET {
			metadata = append(metadata, protocol.IP...)
		}

		//Send the packet
		s.host.GetUDP().SendDown(data, destAddr, metadata, nil)
	}

	if s.sockType == TCP {
		s.Send(data)
	}
}

func (s *Socket) Send(data []byte) {
	if s.sockType == TCP {
		for _, b := range data {
			s.tcpConnection.Send(b)
		}
	}
}

func (s *Socket) Recv(maxBytes int) []byte {
	if s.sockType == UDP {
		if s.udpBinding == nil {
			log.Printf("Socket: Not bound to any address")
			return nil
		}

		if len(s.data) == maxBytes {
			data := make([]byte, maxBytes)
			copy(data, s.data)
			s.data = nil
			return data
		}

		if len(s.data) > maxBytes {
			data := make([]byte, maxBytes)
			copy(data, s.data)
			s.data = s.data[maxBytes:]
			return data
		}

		var data []byte
		for {
			d := s.udpBinding.Recv()
			if d == nil {
				data = append(s.data, data...)
				s.data = nil
				return data
			}

			s.data = append(s.data, d...)

			if len(s.data) == maxBytes {
				data := make([]byte, maxBytes)
				copy(data, s.data)
				s.data = nil
				return data
			}

			if len(s.data) > maxBytes {
				data := make([]byte, maxBytes)
				copy(data, s.data)
				s.data = s.data[maxBytes:]
				return data
			}
		}
	}

	if s.sockType == TCP {
		data := make([]byte, 0, maxBytes)
		for len(data) < maxBytes {
			b := s.tcpConnection.Recv()
			if b == nil {
				break
			}
			data = append(data, *b)
		}
		return data
	}

	return nil
}

func (s *Socket) Close() {
	if s.sockType == UDP {
		s.udpBinding.Close()
	}

	if s.sockType == TCP {
		s.tcpConnection.Close()
	}
}

/*
Internal methods
*/
func (s *Socket) getRandomPort() uint16 {
	for {
		port := uint16(rand.Intn(65536))
		if s.sockType == UDP {
			if !s.host.GetUDP().IsPortInUse(port) {
				return port
			}
		}
		if s.sockType == TCP {
			if !s.host.GetTCP().IsPortInUse(port) {
				return port
			}
		}
	}
}
