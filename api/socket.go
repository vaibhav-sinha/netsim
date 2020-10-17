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
	SOCK_DGRAM = 0
)

// Socket type
const (
	UDP = 0
)

/*
Socket is a high level API that abstracts the protocol being used and offers a uniform interface for applications to
talk to the network. Though it is a (very) leaky abstraction, but still very useful.
*/
type Socket struct {
	host        Host
	sockType    int
	networkType int
	binding     *l4.UdpBinding
	data        []byte
}

func NewSocket(host Host, domain int, channelType int, protocol int) *Socket {
	s := &Socket{
		host:        host,
		networkType: domain,
	}

	if channelType == SOCK_DGRAM && protocol == 0 {
		s.sockType = UDP
	}

	return s
}

/*
The Socket API
*/
func (s *Socket) Bind(ipAddr []byte, port uint16) {
	if s.sockType == UDP {
		s.binding = s.host.GetUDP().Bind(ipAddr, port)
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
}

func (s *Socket) Recv(maxBytes int) []byte {
	if s.sockType == UDP {
		if s.binding == nil {
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
			d := s.binding.Recv()
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

	return nil
}

func (s *Socket) Close() {
	if s.sockType == UDP {
		s.binding.Close()
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
	}
}
