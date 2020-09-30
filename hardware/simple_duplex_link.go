package hardware

type SimpleDuplexLink struct {
	link1 *SimpleLink
	link2 *SimpleLink
}

func NewSimpleDuplexLink(length int64, dataRate int64, byteErrorRate float32, adapter1 Adapter, adapter2 Adapter) *SimpleDuplexLink {
	link := &SimpleDuplexLink{
		link1: NewSimpleLink(length, dataRate, byteErrorRate, adapter1, adapter2),
		link2: NewSimpleLink(length, dataRate, byteErrorRate, adapter2, adapter1),
	}

	RegisterClockConsumer(link)
	return link
}

func (l *SimpleDuplexLink) ClockTrigger() {
	go l.link1.ClockTrigger()
	go l.link2.ClockTrigger()
}
