package hardware

import (
	"sync"
	"time"
)

const (
	TimeSlowdownFactor = int64(1e6)
	ClockRate          = int64(1e9)
)

var (
	Clk  *clock
	once sync.Once
)

type clock struct {
	counter   int64
	rate      int64 //Hertz
	ticker    *time.Ticker
	consumers []clockConsumer
	lock      sync.Mutex
}

type clockConsumer interface {
	ClockTrigger()
}

func RegisterClockConsumer(consumer clockConsumer) {
	once.Do(func() {
		Clk = &clock{
			counter: 0,
			rate:    ClockRate,
			ticker:  time.NewTicker(time.Duration(TimeSlowdownFactor*1000/ClockRate) * time.Millisecond),
		}
	})

	Clk.lock.Lock()
	defer Clk.lock.Unlock()

	Clk.consumers = append(Clk.consumers, consumer)
}

func GetTick() int64 {
	Clk.lock.Lock()
	defer Clk.lock.Unlock()
	return Clk.counter
}

func (*clock) Start() {
	for {
		select {
		case <-Clk.ticker.C:
			Clk.lock.Lock()
			Clk.counter += 1
			Clk.lock.Unlock()
			for _, c := range Clk.consumers {
				go c.ClockTrigger()
			}
		}
	}
}
