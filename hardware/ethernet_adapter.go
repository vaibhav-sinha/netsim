package hardware

import (
	"sync"
)

/*
An ethernet adapter would use the ethernet protocol and have buffers for holding the packets
*/
const (
	readBufferSize = 1000
)

type EthernetAdapter struct {
	readBuffer      chan byte
	writeBuffer     []byte
	macAddress      []byte
	promiscuousMode bool
	isOn            bool
	lock            sync.Mutex
}

/*
Constructor
*/
func NewEthernetAdapter(macAddress []byte, promiscuousMode bool) *EthernetAdapter {
	return &EthernetAdapter{
		readBuffer:      make(chan byte, readBufferSize),
		macAddress:      macAddress,
		promiscuousMode: promiscuousMode,
		isOn:            false,
	}
}

/*
Following methods make this an adapter
*/
func (e *EthernetAdapter) GetByte() *byte {
	e.lock.Lock()
	defer e.lock.Unlock()

	if !e.isOn {
		return nil
	}

	if len(e.writeBuffer) > 0 {
		b := e.writeBuffer[0]
		e.writeBuffer = e.writeBuffer[1:len(e.writeBuffer)]
		return &b
	}

	return nil
}

func (e *EthernetAdapter) SetByte(b byte) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if !e.isOn {
		return
	}

	e.readBuffer <- b
}

func (e *EthernetAdapter) PutInBuffer(b []byte) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if !e.isOn {
		return
	}

	e.writeBuffer = append(e.writeBuffer, b...)
}

func (e *EthernetAdapter) TurnOn() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.isOn = true
}

func (e *EthernetAdapter) TurnOff() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.isOn = false
	e.readBuffer = make(chan byte, readBufferSize)
	e.writeBuffer = nil
}

/*
Methods to expose settings
*/
func (e *EthernetAdapter) GetMacAddress() []byte {
	return e.macAddress
}

func (e *EthernetAdapter) IsPromiscuous() bool {
	return e.promiscuousMode
}

func (e *EthernetAdapter) GetReadBuffer() chan byte {
	return e.readBuffer
}
