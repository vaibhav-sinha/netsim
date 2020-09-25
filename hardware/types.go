package hardware

type Adapter interface {
	GetByte() *byte
	SetByte(byte)
	PutInBuffer([]byte)
	TurnOn()
	TurnOff()
}
