package l2

import (
	"log"
	"netsim/hardware"
	"netsim/protocol"
	"netsim/utils"
)

/*
We will do a simple implementation of ethernet-like protocol which only caters to point-to-point links and hence does
not deal with carrier sensing.

Frame format:

Preamble  - 8 bytes
Dest addr - 6 bytes
Src addr  - 6 bytes
VLAN Id   - 2 bytes
Type      - 2 bytes
Body      - No fixed length
Checksum  - checksumLength bytes
*/

var (
	checksumLength = 1
	mtu            = 1500
	broadcastAddr  = utils.HexStringToBytes("FFFFFFFFFFFF")
	multicastAddr  = utils.HexStringToBytes("01005E")
	defaultVlanId  = utils.HexStringToBytes("0000")
)

type Ethernet struct {
	buffer      []byte
	preamble    []byte
	adapter     *hardware.EthernetAdapter
	l3Protocols []protocol.L3Protocol
	rawConsumer protocol.FrameConsumer
}

/*
Constructor
*/
func NewEthernet(adapter *hardware.EthernetAdapter, rawConsumer protocol.FrameConsumer) *Ethernet {
	s := &Ethernet{
		preamble:    []byte("01020304"),
		adapter:     adapter,
		rawConsumer: rawConsumer,
	}

	go s.run()
	return s
}

/*
Next 3 methods make this an implementation of Protocol
*/
func (s *Ethernet) GetIdentifier() []byte {
	//At L2, there are no identifiers. The protocol is fixed for a particular kind of adapter, hence there is no need of a de-multiplexing key
	return nil
}

func (s *Ethernet) SendDown(data []byte, destAddr []byte, metadata []byte, l3Protocol protocol.Protocol) {
	b := []byte{}
	b = append(b, s.preamble...)
	b = append(b, destAddr...)
	b = append(b, s.adapter.GetMacAddress()...)
	b = append(b, defaultVlanId...)
	b = append(b, l3Protocol.GetIdentifier()...)
	b = append(b, data...)
	b = append(b, utils.CalculateChecksum(b)...)
	s.adapter.PutInBuffer(b)
}

func (s *Ethernet) SendUp([]byte, []byte, protocol.Protocol) {
	//Not used since at L2 level the adapter sends the data up byte-by-byte
}

/*
Next 2 methods make this an implementation of L2Protocol
*/
func (s *Ethernet) GetMTU() int {
	return mtu
}

func (s *Ethernet) GetAdapter() hardware.Adapter {
	return s.adapter
}

func (s *Ethernet) AddL3Protocol(l3Protocol protocol.L3Protocol) {
	s.l3Protocols = append(s.l3Protocols, l3Protocol)
}

/*
Internal methods
*/
func (s *Ethernet) setByte(b *byte) {
	if b == nil {
		s.checkForFrame()
	} else {
		s.buffer = append(s.buffer, *b)
	}
}

func (s *Ethernet) checkForFrame() {
	if len(s.buffer) == 0 {
		return
	}

	if len(s.buffer) < 24+checksumLength {
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
				log.Printf("Ethernet: mac %s: Got frame destined to someone else. Dropping.", string(s.adapter.GetMacAddress()))
				//Remove previous frame from buffer
				s.buffer = nil
				return
			}
		}

		if s.rawConsumer != nil {
			s.rawConsumer.SendUp(previousFrame[:], nil, s)
		}

		if len(s.l3Protocols) > 0 {
			frameType := previousFrame[22:24]
			var upperLayerProtocol protocol.Protocol
			for _, p := range s.l3Protocols {
				identifier := p.GetIdentifier()
				if frameType[0] == identifier[0] && frameType[1] == identifier[1] {
					upperLayerProtocol = p
					break
				}
			}
			if upperLayerProtocol != nil {
				previousFrame = previousFrame[24:]
				previousFrame = previousFrame[:len(previousFrame)-checksumLength]
				upperLayerProtocol.SendUp(previousFrame, nil, s)
			} else {
				log.Printf("Ethernet: mac %s: Got unrecognized frame type: %v", string(s.adapter.GetMacAddress()), frameType)
			}
		}
	} else {
		log.Printf("Ethernet: Got corrupted frame")
	}

	//Remove previous frame from buffer
	s.buffer = nil
}

func (s *Ethernet) validateChecksum(data []byte) bool {
	calculated := utils.CalculateChecksum(data[:len(data)-checksumLength])
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

func (s *Ethernet) isFrameForMe(destAddr []byte) bool {
	// Is broadcast address
	isBroadcast := true
	for i := 0; i < len(broadcastAddr); i++ {
		if broadcastAddr[i] != destAddr[i] {
			isBroadcast = false
		}
	}

	if isBroadcast {
		return isBroadcast
	}

	// Is multicast address
	isMulticast := true
	for i := 0; i < len(multicastAddr); i++ {
		if multicastAddr[i] != destAddr[i] {
			isMulticast = false
		}
	}

	if isMulticast {
		return isMulticast
	}

	// Is this adapter's address
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

func (s *Ethernet) run() {
	for {
		select {
		case b := <-s.adapter.GetReadBuffer():
			s.setByte(b)
		}
	}
}
