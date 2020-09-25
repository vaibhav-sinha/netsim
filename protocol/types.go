package protocol

type Protocol interface {
	GetIdentifier() []byte
	SendDown(data []byte, destAddr []byte, sender Protocol)
	SendUp([]byte)
}
