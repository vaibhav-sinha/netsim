package utils

/*
ByteArrayBuffer
*/
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

func (c *Buffer) Get(blocking bool) []byte {
	if blocking {
		return <-c.items
	} else {
		select {
		case data := <-c.items:
			return data
		default:
			return nil
		}
	}
}

/*
ByteBuffer
*/
type ByteBuffer struct {
	items chan byte
}

func NewByteBuffer(size int) *ByteBuffer {
	buf := &ByteBuffer{
		items: make(chan byte, size),
	}

	return buf
}

func (c *ByteBuffer) Put(item byte) {
	select {
	case c.items <- item:
		return
	default:
		return
	}
}

func (c *ByteBuffer) Get(blocking bool) *byte {
	if blocking {
		data := <-c.items
		return &data
	} else {
		select {
		case data := <-c.items:
			return &data
		default:
			return nil
		}
	}
}
