package protocol

type FrameConsumer interface {
	SendUp([]byte)
}

type FrameProducer interface {
	SendDown(data []byte, destAddr []byte, sender Protocol)
}

type Protocol interface {
	FrameConsumer
	FrameProducer
	GetIdentifier() []byte
}
