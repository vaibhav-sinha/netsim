package utils

type Buffer struct {
	items chan []byte
}

func NewBuffer(size int) *Buffer {
	buf := &Buffer{
		items: make(chan []byte, size),
	}

	return buf
}

func (c *Buffer) Put(item []byte) {
	select {
	case c.items <- item:
		return
	default:
		return
	}
}

func (c *Buffer) Get() []byte {
	select {
	case data := <-c.items:
		return data
	default:
		return nil
	}
}
