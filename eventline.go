package eventline

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Event interface remains the same
type Event interface {
	GetEventPresentation() string
	GetTimeStamp() time.Time
}

// Modified Events struct to use generics
type Events[T Event] struct {
	GlobalEvents int
	EventsMap    map[string]T
}

// Modified constructor with generics
func NewEventHandler[T Event](ge int, em map[string]T) Events[T] {
	return Events[T]{
		GlobalEvents: ge,
		EventsMap:    em,
	}
}

// Modified accessor methods
func (e Events[T]) GetGlobalEventsNumber() int {
	return e.GlobalEvents
}

func (e Events[T]) GetEventsMap() map[string]T {
	return e.EventsMap
}

func (e Events[T]) GetEventsListSize() int {
	return len(e.EventsMap)
}

const (
	Aphabetical = iota
	Time
)

const (
	Desc = iota
	Asc
)

// Modified EventLine to use generics
type EventLine[T Event] struct {
	screen      tcell.Screen
	dataCh      chan Events[T]
	titleStyle  tcell.Style
	textStyle   tcell.Style
	log         zerolog.Logger
	loggingFile *os.File
	style       int
	order       int
	titleString string
}

func initLoggingFile(fileName string) *os.File {
	file, err := os.OpenFile(
		fileName,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0664,
	)
	if err != nil {
		panic(err)
	}
	return file
}

func LogInint(file *os.File) zerolog.Logger {
	return zerolog.New(file).With().Timestamp().Logger()
}

// NewPresenter initializes and returns a new instance of EventLine with a title and configured tcell screen for display.
func NewPresenter[T Event](title string) *EventLine[T] {
	var (
		err    error
		screen tcell.Screen
	)
	screen, err = tcell.NewScreen()
	if err != nil {
		log.Fatal().Msgf("Error creating tcell screen: %v", err)
	}

	err = screen.Init()
	if err != nil {
		log.Fatal().Msgf("Error initializing tcell: %v", err)
	}

	return &EventLine[T]{
		screen:      screen,
		dataCh:      make(chan Events[T], 5),
		titleStyle:  tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true),
		textStyle:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		loggingFile: nil,
		titleString: title,
	}
}

// WithLogFile associates a log file with the EventLine instance for logging operations and initializes the logger.
func (p *EventLine[T]) WithLogFile(fileName string) *EventLine[T] {
	loggingFile := initLoggingFile(fileName)
	p.loggingFile = loggingFile
	p.log = LogInint(loggingFile)
	return p
}

// AlphabeticalOrdered sets the sorting style of the EventLine to alphabetical order and returns the updated instance.
func (p *EventLine[T]) AlphabeticalOrdered() *EventLine[T] {
	p.style = Aphabetical
	return p
}

// TimedOrdered configures the EventLine to sort events based on their timestamp and the specified order.
func (p *EventLine[T]) TimedOrdered(order int) *EventLine[T] {
	p.style = Time
	p.order = order
	return p
}

// Start initializes the EventLine's main loop, handling data processing, screen updates, and context cancellation.
func (p *EventLine[T]) Start(cancel func(), ctx context.Context) {
	var (
		eventCount int
		alarmCount int
	)

	go p.listenKey(cancel)

	p.log.Info().Msg("Starting")

	for {
		// clear the screen
		p.screen.Clear()
		p.title(alarmCount, eventCount)

		select {
		case data := <-p.dataCh:
			p.log.Info().Msgf("Received data size: %d", data.GetEventsListSize())
			dataString := p.applyStyle(data.GetEventsMap())
			p.displayMap(dataString)
			eventCount = data.GetEventsListSize()
			alarmCount = data.GetGlobalEventsNumber()
		case <-time.After(5 * time.Second):
			p.log.Info().Msg("Timeout occurred, no data received")
		case <-ctx.Done():
			p.log.Info().Msg("Context Done")
			return
		default:
			p.log.Debug().Msg("No data available, skipping")
		}

		// Update screen
		p.screen.Show()
		time.Sleep(1 * time.Second)
	}
}

// applyStyle sorts and processes the input events map into a slice of strings based on the configured style and order.
func (p *EventLine[T]) applyStyle(events map[string]T) []string {
	switch p.style {
	case Aphabetical:
		return sortMapByKey(events)
	case Time:
		p.log.Debug().Msgf("events: %+v", events)
		return sortMapByTime(events, p.order)
	default:
		return sortMapByKey(events)
	}
}

// listenKey handles key press events such as application exit, screen sync, or screen clear based on the pressed key.
func (p *EventLine[T]) listenKey(cancel func()) {
	for {
		// Poll event
		evP := p.screen.PollEvent()

		switch ev := evP.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				// closing everything
				p.screen.Fini()
				p.log.Info().Msg("Exiting")
				p.loggingFile.Close()
				cancel()
				return
			} else if ev.Key() == tcell.KeyCtrlL {
				p.screen.Sync()
			} else if ev.Rune() == 'C' || ev.Rune() == 'c' {
				p.screen.Clear()
			}
		}
	}
}

func (p *EventLine[T]) Stop() {
}

// Send publishes the given event data through the EventLine's internal channel for further processing and display.
func (p *EventLine[T]) Send(data Events[T]) {
	p.log.Debug().Msg("Sending data")
	if p.dataCh != nil {
		if len(p.dataCh) < 5 {
			p.dataCh <- data
			p.log.Debug().Msg("Channel successful send data")
			return
		}
		p.log.Debug().Msg("Channel is full")
		return
	}
	p.log.Debug().Msg("Channel is nil")
}

// title updates the display header with the title string, current time, global event counter, and event count.
func (p *EventLine[T]) title(counter int, trucks int) {
	// Current time
	currentTime := time.Now().Local()
	// Format the time up to the second
	formattedTime := currentTime.Format("2006-01-02 15:04:05")
	writeText(p.screen, 0, 0, p.titleStyle, p.titleString)
	writeText(p.screen, 0, 1, p.titleStyle, fmt.Sprintf("Time: %-24s -- Global Counter: %-4d", formattedTime, counter))
	writeText(p.screen, 31, 2, p.titleStyle, fmt.Sprintf("--    Event Count: %-4d", trucks))
}

// debug renders a debug message on the screen at a fixed position using the specified style.
func (p *EventLine[T]) debug(msg string) {
	writeText(p.screen, 0, 3, p.titleStyle, msg)
}

// displayMap renders the provided data slice onto the screen, starting from a fixed row position with a specific style.
func (p *EventLine[T]) displayMap(data []string) {
	// Display the data in the map
	row := 4
	dataStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	for _, truck := range data {
		writeText(p.screen, 0, row, dataStyle, truck)
		row++
	}
}

// Modified sorting functions
func DebugSortMapByTime[T Event](data map[string]T, order int) []string {
	return sortMapByTime(data, order)
}

func writeText(screen tcell.Screen, x, y int, style tcell.Style, text string) {
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

// sortMapByKey sorts a map by its string keys in ascending order and returns a slice of event presentations as strings.
func sortMapByKey[T Event](data map[string]T) []string {
	// Get all keys from the map
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	// Sort the keys
	sort.Strings(keys)
	// Build the ordered result
	result := make([]string, 0, len(data))
	for _, key := range keys {
		result = append(result, data[key].GetEventPresentation())
	}
	return result
}

// sortMapByTime sorts the input map of events by timestamp in ascending or descending order based on the order parameter.
// Returns a slice of event presentations ordered by the specified criteria.
func sortMapByTime[T Event](data map[string]T, order int) []string {
	// result and timeStamps are built in order, all elements in them are ordered based on the criteria "order"
	result := make([]string, len(data))
	timeStamps := make([]int64, len(data))
	index := 0
	posToInsert := 0
	for key, e := range data {
		posToInsert = index
		index++
		v := e.GetTimeStamp().UnixMilli()
	loop:
		for i := 0; i < index-1; i++ {
			if compareUsing(v, timeStamps[i], order) {
				// condition is true
				// we are shifting all element in timeStamps and result starting at position i to the
				// last element we have (index-1) in these 2 lists
				shitDownFromIndex(timeStamps, result, index-1, i)
				posToInsert = i
				break loop
			}
		}
		timeStamps[posToInsert] = v
		result[posToInsert] = data[key].GetEventPresentation()
	}
	return result
}

// shitDownFromIndex shifts elements in two slices from the start index down to the stop index.
func shitDownFromIndex(t []int64, s []string, start int, stop int) {
	for i := start; i > stop; i-- {
		t[i] = t[i-1]
		s[i] = s[i-1]
	}
}

func compareUsing(a int64, b int64, order int) bool {
	switch order {
	case Desc:
		return a > b
	case Asc:
		return a < b
	default:
		return false
	}
}
