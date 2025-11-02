package synchronized

import (
	"sync"

	lcd "github.com/hardcodead/go-pi-lcd1602"
	"github.com/hardcodead/go-pi-lcd1602/animations"
)

type SynchronizedLCD struct {
	lcd.LCDI
	line1, line2 sync.Mutex
}

func NewSynchronizedLCD(l lcd.LCDI) *SynchronizedLCD {
	l.Initialize()
	return &SynchronizedLCD{
		l, sync.Mutex{}, sync.Mutex{},
	}
}

func (l *SynchronizedLCD) WriteLines(lines ...string) {
	if len(lines) > 0 {
		l.line1.Lock()
		l.WriteLine(lines[0], lcd.Line1)
		l.line1.Unlock()
	}
	if len(lines) > 1 {
		l.line2.Lock()
		l.WriteLine(lines[1], lcd.Line2)
		l.line2.Unlock()
	}
}

func (l *SynchronizedLCD) Animate(animation animations.Animation, line lcd.LineNumber) chan bool {
	done := make(chan bool, 1)

	switch line {
	case lcd.Line1:
		l.line1.Lock()
	case lcd.Line2:
		l.line2.Lock()
	}

	go func() {
		animation.Width(l.Width())
		for !animation.Done() {
			s := animation.Content()
			l.WriteLine(s, line)
			animation.Delay()

		}

		switch line {
		case lcd.Line1:
			l.line1.Unlock()
		case lcd.Line2:
			l.line2.Unlock()
		}
		done <- true
	}()

	return done
}
