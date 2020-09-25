package hardware

type Link struct {
	dataRate     int64    //bits per second
	bitErrorRate float32  //fraction of corrupted bits
	speedOfLight int64    //metres per second
	length       float64  //metres
	nodes        []node   //bit streams
	pulses       []*pulse //data on the link
	locations    []bool   //is a location occupied on the link
}

type node interface {
	getLocation() float64
	getBit() *uint8
	setBit(uint8)
}

type pulse struct {
	value     uint8
	position  int
	direction bool
	expired   bool
}

func NewCopperLink(length float64, nodes []node) *Link {
	link := &Link{
		dataRate:     10e6,
		bitErrorRate: 0.01,
		speedOfLight: 2.3e8,
		length:       length,
		nodes:        nodes,
	}

	numberOfLocations := int(length*float64(ClockRate)/float64(link.speedOfLight)) + 1
	for i := 0; i < numberOfLocations; i++ {
		link.locations = append(link.locations, false)
	}

	RegisterClockConsumer(link)
	return link
}

func (l *Link) ClockTrigger() {
	// Move the pulses
	for _, p := range l.pulses {
		if p.direction {
			l.locations[p.position] = false
			p.position += 1
			if p.position >= len(l.locations) {
				p.expired = true
			} else {
				if l.locations[p.position] {
					l.reportCollision()
				} else {
					l.locations[p.position] = true
					//If a node is at this position, send it the bit value
				}
			}
		} else {
			l.locations[p.position] = false
			p.position -= 1
			if p.position < 0 {
				p.expired = true
			} else {
				if l.locations[p.position] {
					l.reportCollision()
				} else {
					l.locations[p.position] = true
				}
			}
		}
	}

	pulses := []*pulse{}
	for _, p := range l.pulses {
		if !p.expired {
			pulses = append(pulses, p)
		}
	}
	l.pulses = pulses

	// Add new pulses

}

func (l *Link) reportCollision() {

}
