//----------------------------------------------------------------------------------------------------------------------
//       eventline.go
//----------------------------------------------------------------------------------------------------------------------

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

const (
	refreshRate = 200 * time.Millisecond
	titleHeight = 4
)

// Event represents an interface for an event with methods for presentation and timestamp retrieval.
// GetEventPresentation provides a formatted string representation of the event.
// GetTimeStamp retrieves the timestamp associated with the event.
type Event interface {
	GetEventPresentation() string
	GetTimeStamp() time.Time
}

// Events represents a collection of event data supporting generics for flexible use with various event types.
// GlobalEvents tracks the total number of events.
// LastUpdate holds the identifier of the most recent update.
// EventsMap maps string keys to events of type T, providing efficient access to event data.
type Events[T Event] struct {
	GlobalEvents int
	LastUpdate   string
	EventsMap    map[string]T
}

// NewEventHandler initializes and returns a new Events instance with the provided global event count, event map, and last update ID.
func NewEventHandler[T Event](ge int, em map[string]T, lastId string) Events[T] {
	return Events[T]{
		GlobalEvents: ge,
		EventsMap:    em,
		LastUpdate:   lastId,
	}
}

// GetGlobalEventsNumber returns the total number of global events stored in the Events collection.
func (e Events[T]) GetGlobalEventsNumber() int {
	return e.GlobalEvents
}

// GetEventsMap returns a shallow copy of the EventsMap, ensuring the original map remains unaffected by modifications.
func (e Events[T]) GetEventsMap() map[string]T {
	return e.EventsMap
}

// GetEventsListSize returns the number of events currently stored in the EventsMap.
func (e Events[T]) GetEventsListSize() int {
	return len(e.EventsMap)
}

// GetLastUpdateID retrieves the identifier of the most recent update from the Events collection.
func (e Events[T]) GetLastUpdateID() string {
	return e.LastUpdate
}

// DeepCopy creates a deep copy of the provided Events[T] instance, ensuring all contained data is duplicated.
func DeepCopy[T Event](e Events[T]) Events[T] {
	// Create a new map with the same capacity
	copyMap := make(map[string]T, len(e.EventsMap))

	// Copy all key-value pairs
	for key, value := range e.EventsMap {
		copyMap[key] = value
	}
	return Events[T]{
		GlobalEvents: e.GlobalEvents,
		EventsMap:    copyMap,
		LastUpdate:   e.LastUpdate,
	}
}

const (
	Aphabetical = iota
	Time
)

const (
	Desc = iota
	Asc
)

// EventLine represents a processing pipeline for rendering events with various styles and configurations.
type EventLine[T Event] struct {
	screen       tcell.Screen
	dataCh       chan Events[T]
	titleStyle   tcell.Style
	textStyle    tcell.Style
	highlight    tcell.Style
	positionText tcell.Style
	log          zerolog.Logger
	loggingFile  *os.File
	style        int
	order        int
	titleString  string
}

// initLoggingFile creates or opens a file for logging with the specified name in append, create, and write modes.
// It returns a pointer to the opened file or panics if an error occurs.
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

// LogInint initializes and returns a new zerolog.Logger with timestamping using the provided file for output.
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
		screen:       screen,
		dataCh:       make(chan Events[T], 50),
		titleStyle:   tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true),
		textStyle:    tcell.StyleDefault.Foreground(tcell.ColorWhite),
		highlight:    tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite),
		positionText: tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorRed),
		loggingFile:  nil,
		titleString:  title,
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
		eventCount      int
		alarmCount      int
		dataString      []string
		keyChan         chan *tcell.EventKey
		position        int
		userPosition    int
		rows            int
		ticker          time.Ticker
		lastUpdateIndex int
	)

	keyChan = make(chan *tcell.EventKey)
	go p.listenKey(cancel, keyChan)

	p.log.Info().Msg("Starting")

	_, rows = p.screen.Size()

	ticker = time.Ticker{C: time.Tick(5 * time.Second)}
	refreshTicker := time.NewTicker(refreshRate)
	// clear the screen
	p.screen.Clear()
	p.title(alarmCount, eventCount)

	for {
		select {
		case data := <-p.dataCh:
			p.log.Info().Msgf("Received data size: %d", data.GetEventsListSize())
			dataString, lastUpdateIndex = p.applyStyle(data.GetEventsMap(), data.GetLastUpdateID())
			eventCount = data.GetEventsListSize()
			alarmCount = data.GetGlobalEventsNumber()
		case <-ticker.C:
			p.log.Debug().
				Int("position", position).
				Int("userPosition", userPosition).
				Int("maxLine", min(rows-titleHeight, eventCount)).
				Int("rows", rows).
				Int("dataLen", eventCount).
				Msg("displayMap")
		case <-ctx.Done():
			p.log.Info().Msg("Context Done")
			ticker.Stop()
			return
		case ev := <-keyChan:
			p.log.Debug().Msgf("Key event: %v", ev)
			p.log.Debug().Msgf("Key: %v", ev.Key())

			switch ev.Key() {
			case tcell.KeyUp:
				if userPosition >= 1 {
					userPosition--
					break
				}
				if position >= 1 {
					position--
				}
			case tcell.KeyDown:
				// we don't want to go above the number of rows we have on the screen or
				// the number of events if we have fewer events than rows
				if userPosition < rows-1-titleHeight && userPosition < eventCount-1 {
					userPosition++
					break
				}
				if position < eventCount-rows {
					position++
				}
			default:
				// we ignore other key press
			}
			p.Refresh(alarmCount, eventCount, dataString, position, userPosition, lastUpdateIndex)

		case <-refreshTicker.C:
			p.Refresh(alarmCount, eventCount, dataString, position, userPosition, lastUpdateIndex)

		default:

		}
	}
}

// Refresh clears the screen, updates the title, and renders event data at specified positions with highlighting.
func (p *EventLine[T]) Refresh(alrmCount int, eventCount int, dataString []string, position int, userPosition int, lastUpdateIndex int) {
	p.screen.Clear()
	p.title(alrmCount, eventCount)
	p.displayMap(dataString, position, userPosition, lastUpdateIndex)
	p.screen.Show()
}

// Close releases resources associated with the screen and log file, ensuring proper cleanup and termination.
func (p *EventLine[T]) Close() {
	p.screen.Fini()
	p.log.Info().Msg("Exiting!")
	if p.loggingFile != nil {
		err := p.loggingFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
}

// applyStyle sorts and processes the input events map into a slice of strings based on the configured style and order.
func (p *EventLine[T]) applyStyle(events map[string]T, lastUpdate string) ([]string, int) {
	switch p.style {
	case Aphabetical:
		return sortMapByKey(events, lastUpdate)
	case Time:
		//p.log.Debug().Msgf("events: %+v", events)
		return sortMapByTime(events, p.order, lastUpdate)
	default:
		return sortMapByKey(events, lastUpdate)
	}
}

// listenKey handles key press events such as application exit, screen sync, or screen clear based on the pressed key.
func (p *EventLine[T]) listenKey(cancel func(), keyCahn chan *tcell.EventKey) {
	for {
		// Poll event
		evP := p.screen.PollEvent()

		switch ev := evP.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				cancel()
				return
			} else if ev.Key() == tcell.KeyCtrlL {
				p.screen.Sync()
			} else if ev.Rune() == 'C' || ev.Rune() == 'c' {
				p.screen.Clear()
			} else {
				keyCahn <- ev
			}
		}
	}
}

// Stop terminates the EventLine instance by performing cleanup operations and releasing associated resources.
func (p *EventLine[T]) Stop() {
	p.Close()
}

// Send publishes the given event data through the EventLine's internal channel for further processing and display.
func (p *EventLine[T]) Send(data Events[T]) {
	p.log.Debug().Msg("Sending data")
	if p.dataCh != nil {
		var dataToSend Events[T]
		dataToSend = DeepCopy(data)
		select {
		case p.dataCh <- dataToSend:
			p.log.Debug().Msg("Channel successful send data")
			return
		default:
			p.log.Debug().Msg("Channel is full")
			return
		}
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

// displayMap renders the provided data slice onto the screen, starting from a fixed row position with a specific style.
func (p *EventLine[T]) displayMap(data []string, position int, userPosition int, index int) {

	_, rows := p.screen.Size()
	dataLen := len(data)
	// Display the data in the map
	row := 4

	// Find the end
	maxLine := min(rows-titleHeight, dataLen)

	for i := position; i < maxLine+position; i++ {
		switch i {
		case index:
			writeText(p.screen, 0, row, p.highlight, data[i])
		case userPosition + position:
			// userPosition is the screen, position in the list, row is the line we are drawing
			writeText(p.screen, 0, row, p.positionText, data[i])
		default:
			writeText(p.screen, 0, row, p.textStyle, data[i])
		}
		row++
	}
}

// debug renders a debug message on the screen at a fixed position using the specified style.
func (p *EventLine[T]) debug(msg string) {
	writeText(p.screen, 0, 3, p.titleStyle, msg)
}

//----------------------------------------------------------------------------------------------------------------------
//       Helper Functions
//----------------------------------------------------------------------------------------------------------------------

// DebugSortMapByTime sorts the provided map of events by timestamp in ascending or descending order, based on the order parameter.
// It returns a slice of formatted event presentations and the index of the last updated event, if applicable.
func DebugSortMapByTime[T Event](data map[string]T, order int) ([]string, int) {
	return sortMapByTime(data, order, "")
}

func writeText(screen tcell.Screen, x, y int, style tcell.Style, text string) {
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

// sortMapByKey sorts a map by its string keys in ascending order and returns a slice of event presentations as strings.
func sortMapByKey[T Event](data map[string]T, lastUpdate string) ([]string, int) {
	var lastIndexUpdate int

	// Get all keys from the map
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	// Sort the keys
	sort.Strings(keys)

	// Build the ordered result
	result := make([]string, len(data))

	for i, key := range keys {
		result[i] = data[key].GetEventPresentation()
		if lastUpdate == key {
			lastIndexUpdate = i
		}
	}
	return result, lastIndexUpdate
}

// sortMapByTime sorts the input map of events by timestamp in ascending or descending order based on the order parameter.
// Returns a slice of event presentations ordered by the specified criteria.
func sortMapByTime[T Event](data map[string]T, order int, lastUpdate string) ([]string, int) {

	// result and timeStamps are built in order, all elements in them are ordered based on the criteria "order"
	result := make([]string, len(data))
	timeStamps := make([]int64, len(data))
	ids := make([]string, len(data))
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
				shiftDownFromIndex(timeStamps, result, ids, index-1, i)
				posToInsert = i
				break loop
			}
		}
		timeStamps[posToInsert] = v
		result[posToInsert] = data[key].GetEventPresentation()
		ids[posToInsert] = key
	}

	for i, v := range ids {
		// ids are unique (they are the map key)
		if v == lastUpdate {
			return result, i
		}
	}
	return result, 0
}

// shiftDownFromIndex shifts elements in two slices from the start index down to the stop index.
func shiftDownFromIndex(t []int64, s []string, id []string, start int, stop int) {
	for i := start; i > stop; i-- {
		t[i] = t[i-1]
		s[i] = s[i-1]
		id[i] = id[i-1]
	}
}

func compareUsing(a int64, b int64, order int) bool {
	switch order {
	case Desc:
		return a >= b
	case Asc:
		return a < b
	default:
		return false
	}
}
