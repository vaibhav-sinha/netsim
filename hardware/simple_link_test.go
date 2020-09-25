package hardware

import (
	"log"
	"testing"
	"time"
)

type TestAdapter struct {
}

func (a *TestAdapter) GetByte() *byte {
	b := byte(GetTick())
	log.Printf("Tick %d: Sending byte %v", GetTick(), b)
	return &b
}

func (a *TestAdapter) SetByte(b *byte) {
	log.Printf("Tick %d: Received byte %v", GetTick(), *b)
}

func (a *TestAdapter) PutInBuffer(b []byte) {

}

func (a *TestAdapter) TurnOn() {

}

func (a *TestAdapter) TurnOff() {

}

func TestBasicDataTransfer(t *testing.T) {
	adapter1 := &TestAdapter{}
	adapter2 := &TestAdapter{}
	NewSimpleLink(100, 1e6, 0.01, adapter1, adapter2)
	go Clk.Start()
	time.Sleep(100 * time.Second)
}
