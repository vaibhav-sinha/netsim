package hardware

type DuplexLink struct {
	link1 *Link
	link2 *Link
}

func NewDuplexLink(length int64, dataRate int64, byteErrorRate float32, adapter1 Adapter, adapter2 Adapter) *DuplexLink {
	link := &DuplexLink{
		link1: NewLink(length, dataRate, byteErrorRate, adapter1, adapter2),
		link2: NewLink(length, dataRate, byteErrorRate, adapter2, adapter1),
	}

	RegisterClockConsumer(link)
	return link
}

func (l *DuplexLink) ClockTrigger() {
	go l.link1.ClockTrigger()
	go l.link2.ClockTrigger()
}
