package l2

import (
	"log"
	"netsim/hardware"
	"netsim/protocol"
)

/*
We will do a simple implementation of ethernet-like protocol which only caters to point-to-point links and hence does
not deal with carrier sensing.

Frame format:

Preamble  - 8 bytes
Dest addr - 6 bytes
Src addr  - 6 bytes
Type      - 2 bytes
Body      - No fixed length
Checksum  - checksumLength bytes
*/

const (
	checksumLength = 1
)

type SimpleEthernet struct {
	buffer      []byte
	identifier  []byte
	preamble    []byte
	adapter     *hardware.EthernetAdapter
	l3Protocols []protocol.Protocol
	rawConsumer protocol.FrameConsumer
}

/*
Constructor
*/
func NewSimpleEthernet(adapter *hardware.EthernetAdapter, l3Protocols []protocol.Protocol, rawConsumer protocol.FrameConsumer) *SimpleEthernet {
	s := &SimpleEthernet{
		identifier:  []byte("00"),
		preamble:    []byte("01020304"),
		adapter:     adapter,
		l3Protocols: l3Protocols,
		rawConsumer: rawConsumer,
	}

	go s.run()
	return s
}

/*
Next 3 methods make this an implementation of Protocol
*/
func (s *SimpleEthernet) GetIdentifier() []byte {
	return s.identifier
}

func (s *SimpleEthernet) SendDown(data []byte, destAddr []byte, l3Protocol protocol.Protocol) {
	b := []byte{}
	b = append(b, s.preamble...)
	b = append(b, destAddr...)
	b = append(b, s.adapter.GetMacAddress()...)
	b = append(b, l3Protocol.GetIdentifier()...)
	b = append(b, data...)
	b = append(b, s.calculateChecksum(b)...)
	s.adapter.PutInBuffer(b)
}

func (s *SimpleEthernet) SendUp([]byte) {
	//Not used since at L2 level the adapter sends the data up byte-by-byte
}

/*
Expose config
*/
func (s *SimpleEthernet) GetAdapter() *hardware.EthernetAdapter {
	return s.adapter
}

/*
Internal methods
*/
func (s *SimpleEthernet) setByte(b *byte) {
	if b == nil {
		s.checkForFrame()
	} else {
		s.buffer = append(s.buffer, *b)
	}
}

func (s *SimpleEthernet) checkForFrame() {
	if len(s.buffer) == 0 {
		return
	}

	if len(s.buffer) < 22+checksumLength {
		s.buffer = nil
		return
	}

	isMatch := true
	for i, b := range s.preamble {
		if s.buffer[i] != b {
			isMatch = false
			break
		}
	}

	if !isMatch {
		s.buffer = nil
		return
	}

	//Preamble detected
	previousFrame := s.buffer
	isValidFrame := s.validateChecksum(previousFrame)
	if isValidFrame {
		if !s.adapter.IsPromiscuous() {
			isFrameForMe := s.isFrameForMe(previousFrame[8:14])
			if !isFrameForMe {
				log.Printf("SimpleEthernet: mac %s: Got frame destined to someone else. Dropping.", string(s.adapter.GetMacAddress()))
				//Remove previous frame from buffer
				s.buffer = nil
				return
			}
		}

		if s.rawConsumer != nil {
			s.rawConsumer.SendUp(previousFrame[:])
		}

		if len(s.l3Protocols) > 0 {
			frameType := previousFrame[20:22]
			var upperLayerProtocol protocol.Protocol
			for _, p := range s.l3Protocols {
				identifier := p.GetIdentifier()
				if frameType[0] == identifier[0] && frameType[1] == identifier[1] {
					upperLayerProtocol = p
					break
				}
			}
			if upperLayerProtocol != nil {
				previousFrame = previousFrame[22:]
				previousFrame = previousFrame[:len(previousFrame)-checksumLength]
				upperLayerProtocol.SendUp(previousFrame)
			} else {
				log.Printf("SimpleEthernet: mac %s: Got unrecognized frame type: %v", string(s.adapter.GetMacAddress()), frameType)
			}
		}
	} else {
		log.Printf("SimpleEthernet: Got corrupted frame")
	}

	//Remove previous frame from buffer
	s.buffer = nil
}

func (s *SimpleEthernet) calculateChecksum(data []byte) []byte {
	checksum := []byte("0")
	for _, d := range data {
		checksum[0] += d
	}

	return checksum
}

func (s *SimpleEthernet) validateChecksum(data []byte) bool {
	calculated := s.calculateChecksum(data[:len(data)-checksumLength])
	actual := data[len(data)-checksumLength:]

	isMatch := true
	for i := 0; i < checksumLength; i++ {
		if actual[i] != calculated[i] {
			isMatch = false
			break
		}
	}
	return isMatch
}

func (s *SimpleEthernet) isFrameForMe(destAddr []byte) bool {
	isMatch := true
	addr := s.adapter.GetMacAddress()

	for i := 0; i < 6; i++ {
		if addr[i] != destAddr[i] {
			isMatch = false
			break
		}
	}

	return isMatch
}

func (s *SimpleEthernet) run() {
	for {
		select {
		case b := <-s.adapter.GetReadBuffer():
			s.setByte(b)
		}
	}
}
