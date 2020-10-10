package protocol

import (
	"netsim/hardware"
)

type FrameConsumer interface {
	SendUp(data []byte, metadata []byte, sender Protocol)
}

type FrameProducer interface {
	SendDown(data []byte, destAddr []byte, metadata []byte, sender Protocol)
}

type Protocol interface {
	FrameConsumer
	FrameProducer
	GetIdentifier() []byte
}

type L2Protocol interface {
	Protocol
	GetMTU() int
	GetAdapter() hardware.Adapter
}

type L3Protocol interface {
	Protocol
	GetAddress() []byte
	SetL2Protocol(L2Protocol)
	GetL2Protocol() L2Protocol
}

type L4Protocol interface {
	Protocol
	SetL3Protocol(L3Protocol)
}

type RouteProvider interface {
	GetGatewayForAddress([]byte) []byte
	GetInterfaceForAddress([]byte) int
}

type AddressResolver interface {
	Resolve([]byte) []byte
}

type CIDR struct {
	Address []byte
	Mask    int
}
