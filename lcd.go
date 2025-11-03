package lcd1602

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	rpio "github.com/stianeikeland/go-rpio"
)

const (
	RSData        = true  // sending data
	RSInstruction = false // sending an instruction

	Line1 = LineNumber(0x80) // address for the 1st line
	Line2 = LineNumber(0xC0) // address for the 2nd line
)

var (
	EnableDelay             = 1 * time.Microsecond
	ExecutionTimeDefault    = 40 * time.Microsecond
	ExecutionTimeReturnHome = 1520 * time.Microsecond
)

// global used to ensure the rpio library is nitialized befure using it..
var rpioPrepared = false

type LineNumber uint8

type Character [8]uint8

type LCD struct {
	RS, E               rpio.Pin
	DataPins            []rpio.Pin
	LineWidth           int
	writelock, linelock sync.Mutex
}

type LCDI interface {
	Initialize()
	ReturnHome()
	EntryModeSet(bool, bool)
	DisplayMode(bool, bool, bool)
	Clear()
	Reset()
	Write(uint8, bool)
	WriteLine(string, LineNumber)
	CreateChar(uint8, Character)
	Width() int
	Close()
}

func SetCustomCharacters(l LCDI, characters []Character) {
	for index, chr := range characters {
		offset := 8 - len(characters) + index
		if offset < 0 {
			continue
		}
		l.CreateChar(uint8(offset), chr)
	}
}

// Open function should be called before executing any other code!
func Open() {
	if err := rpio.Open(); err != nil {
		log.Fatalln(err)
	}

	rpioPrepared = true
}

func Close() {
	if rpioPrepared {
		err := rpio.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func New(rs, e int, data []int, linewidth int) (*LCD, error) {
	datalength := len(data)
	if datalength != 4 && datalength != 8 {
		return nil, errors.New("LCD requires four or eight datapins")
	}

	datapins := make([]rpio.Pin, 0)

	for _, d := range data {
		datapins = append(datapins, rpio.Pin(d))
	}

	l := &LCD{
		RS:        rpio.Pin(rs),
		E:         rpio.Pin(e),
		DataPins:  datapins,
		LineWidth: linewidth,
	}
	l.initPins()
	return l, nil
}

func (l *LCD) Close() {}
func (l *LCD) Width() int {
	return l.LineWidth
}

// Initialize initiates the LCD
func (l *LCD) Initialize() {
	l.Reset()

	l.EntryModeSet(true, false)
	l.DisplayMode(true, false, false) // Display, Cursor, Blink

	l.Write(0x28, RSInstruction) // 00101000 - Set DDRAM Address
	l.ReturnHome()

	l.Clear() // clear screen
	// init time...
	time.Sleep(10 * time.Millisecond)
}

// ReturnHome function returns the cursor to home
func (l *LCD) ReturnHome() {
	l.Write(0x02, RSInstruction)
	time.Sleep(ExecutionTimeReturnHome)
}

// EntryModeSet function
func (l *LCD) EntryModeSet(increment, shift bool) {
	instruction := uint8(0x04)
	if increment {
		instruction |= 0x02
	}
	if shift {
		instruction |= 0x01
	}
	l.Write(instruction, RSInstruction)
}

// DisplayMode function set the display modes
func (l *LCD) DisplayMode(display, cursor, blink bool) {
	instruction := uint8(0x08)

	if display {
		instruction |= 0x04
	}
	if cursor {
		instruction |= 0x02
	}
	if blink {
		instruction |= 0x01
	}
	l.Write(instruction, RSInstruction)
}

// Clear function clears the screen
func (l *LCD) Clear() {
	l.Write(0x01, RSInstruction)
}

// WriteLine function writes a single line fo text to the LCD
// if line length exceeds the linelength of the LCD, aslice will be used
func (l *LCD) WriteLine(s string, line LineNumber) {
	l.linelock.Lock()
	defer l.linelock.Unlock()
	frmt := fmt.Sprintf("%%%ds", l.LineWidth)
	s = fmt.Sprintf(frmt, s)

	s = s[:l.LineWidth]

	l.Write(uint8(line), RSInstruction)

	for _, c := range s {
		l.Write(uint8(c), RSData)
	}
}

// Write function writes data to the LCD
func (l *LCD) Write(data uint8, mode bool) {
	l.writelock.Lock()
	defer l.writelock.Unlock()

	if mode {
		l.RS.High()
	} else {
		l.RS.Low()
	}

	for _, p := range l.DataPins {
		p.Low()
	}

	if len(l.DataPins) == 4 {
		// ofsetfor highest order bits
		base := uint8(0x10)
		for i, dataPin := range l.DataPins {
			setBitToPin(dataPin, data, base<<uint8(i))
		}
		l.enable(ExecutionTimeDefault)
		// lowest order bits
		base = uint8(0x01)
		for i, dataPin := range l.DataPins {
			setBitToPin(dataPin, data, base<<uint8(i))
		}
	} else {
		// all bits
		base := uint8(0x01)
		for i, dataPin := range l.DataPins {
			setBitToPin(dataPin, data, base<<uint8(i))
		}
	}
	l.enable(ExecutionTimeDefault)
}

func (l *LCD) CreateChar(position uint8, data Character) {
	if position > 7 {
		// error
		return
	}
	l.Write(0x40|(position<<3), false)
	for _, x := range data {
		l.Write(x, true)
	}
}

// Reset resets the lcd
func (l *LCD) Reset() {
	// init sequence
	l.Write(0x33, RSInstruction)
	time.Sleep(ExecutionTimeDefault)
	l.Write(0x32, RSInstruction)
	time.Sleep(ExecutionTimeDefault)
}

// setBitToPin function sets given pin to a bit value from a given data int
func setBitToPin(pin rpio.Pin, data, position uint8) {
	if data&position == position {
		pin.High()
	} else {
		pin.Low()
	}
}

// Enable function sets the 'Enable'-pin high, and low to enable 2Xa single write sequence
func (l *LCD) enable(executionTime time.Duration) {
	time.Sleep(EnableDelay)
	l.E.High()
	time.Sleep(EnableDelay)
	l.E.Low()
	time.Sleep(executionTime)
}

func (l *LCD) initPins() {
	if !rpioPrepared {
		Open()
	}
	l.RS.Output()
	l.E.Output()
	for _, d := range l.DataPins {
		d.Output()
	}
}
