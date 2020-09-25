package hardware

import (
	"log"
	"math/rand"
)

/*
Simulating a link is hard and is only worthwhile if dealing with link layer protocols with a focus on collisions.
We want to focus more on higher level protocols and full duplex communication links hence don't need complex link implementation.
SimpleLink will make multiple choices to simplify the implementation:
1. The basic unit of data transfer will be byte and not bit
2. It will only be used in point-to-point scenarios
3. There will only be one Adapter acting as source on a link. Hence collisions cannot happen
*/

const (
	simpleLinkSpeedOfLight = 2e8 //metres per second
)

type SimpleLink struct {
	length        int64   //metres
	dataRate      int64   //bytes per second
	byteErrorRate float32 //fraction of corrupted bytes
	source        Adapter
	destination   Adapter
	pulses        []*byte
}

func NewSimpleLink(length int64, dataRate int64, byteErrorRate float32, source Adapter, destination Adapter) *SimpleLink {
	volume := dataRate * length / simpleLinkSpeedOfLight
	if volume < 2 {
		volume = 2
	}
	pulses := make([]*byte, 0)
	var i int64
	for i = 0; i < volume; i++ {
		pulses = append(pulses, nil)
	}
	link := &SimpleLink{
		length:        length,
		dataRate:      dataRate,
		byteErrorRate: byteErrorRate,
		source:        source,
		destination:   destination,
		pulses:        pulses,
	}

	RegisterClockConsumer(link)
	return link
}

func (l *SimpleLink) ClockTrigger() {
	if GetTick()%(ClockRate/l.dataRate) != 0 {
		return
	}

	for i := len(l.pulses) - 1; i > 0; i-- {
		if i == len(l.pulses)-1 {
			if l.pulses[i] != nil {
				random := rand.Float32()
				if random < l.byteErrorRate {
					log.Printf("SimpleLink: Bit corruption")
					corruptedData := *l.pulses[i] ^ 0x80
					l.pulses[i] = &corruptedData
				}
			}
			l.destination.SetByte(l.pulses[i])
		}
		l.pulses[i] = l.pulses[i-1]
	}
	l.pulses[0] = l.source.GetByte()
}
