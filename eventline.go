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

type Events struct {
	GlobalEvents int
	EventsMap    map[string]interface{}
}

func NewEventHandler(ge int, em map[string]interface{}) Events {
	return Events{
		GlobalEvents: ge,
		EventsMap:    em,
	}
}

func (e Events) GetGlobalEventsNumber() int {
	return e.GlobalEvents
}

func (e Events) GetEventsMap() map[string]interface{} {
	return e.EventsMap
}

func (e Events) GetEventsListSize() int {
	return len(e.EventsMap)
}

type Event interface {
	GetEventPresentation() string
	GetTimeStamp() time.Time
}

const (
	Aphabetical = iota
	Time
)

const (
	Desc = iota
	Asc
)

type EventLine struct {
	screen      tcell.Screen
	dataCh      chan Events
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

func NewPresenter(title string) *EventLine {
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

	return &EventLine{
		screen:      screen,
		dataCh:      make(chan Events, 5),
		titleStyle:  tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true),
		textStyle:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		loggingFile: nil,
		titleString: title,
	}
}

func (p *EventLine) WithLogFile(fileName string) *EventLine {
	loggingFile := initLoggingFile(fileName)
	p.loggingFile = loggingFile
	p.log = LogInint(loggingFile)
	return p
}

func (p *EventLine) AlphabeticalOrdered() *EventLine {
	p.style = Aphabetical
	return p
}

func (p *EventLine) TimedOrdered(order int) *EventLine {
	p.style = Time
	p.order = order
	return p
}

func (p *EventLine) Start(cancel func(), ctx context.Context) {
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

func (p *EventLine) applyStyle(events map[string]interface{}) []string {

	switch p.style {
	case Aphabetical:
		return sortMapByKey(events)
	case Time:
		//TODO
		p.log.Debug().Msgf("events: %+v", events)
		return sortMapByTime(events, p.order)
	default:
		//TODO
		return sortMapByKey(events)
	}
}

func (p *EventLine) listenKey(cancel func()) {
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

func (p *EventLine) Stop() {

}

func (p *EventLine) Send(data Events) {
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

func (p *EventLine) title(counter int, trucks int) {
	// Current time
	currentTime := time.Now().Local()
	// Format the time up to the second
	formattedTime := currentTime.Format("2006-01-02 15:04:05")

	writeText(p.screen, 0, 0, p.titleStyle, p.titleString)
	writeText(p.screen, 0, 1, p.titleStyle, fmt.Sprintf("Time: %-24s -- Global Counter: %-4d", formattedTime, counter))
	writeText(p.screen, 31, 2, p.titleStyle, fmt.Sprintf("--    Event Count: %-4d", trucks))
}

func (p *EventLine) debug(msg string) {
	writeText(p.screen, 0, 3, p.titleStyle, msg)
}

func (p *EventLine) displayMap(data []string) {

	// Display the data in the map
	row := 4
	dataStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	for _, truck := range data {
		//writeText(screen, 0, row, dataStyle, fmt.Sprintf("Truck: %s - Alarms: %d", truck, count))
		writeText(p.screen, 0, row, dataStyle, truck)
		row++
	}

}

func DebugSortMapByTime(data map[string]interface{}, order int) []string {
	return sortMapByTime(data, order)
}

func writeText(screen tcell.Screen, x, y int, style tcell.Style, text string) {
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

func sortMapByKey(data map[string]interface{}) []string {
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
		//result = append(result, fmt.Sprintf("Truck: %-9s - Alarms: %4d - Last alarm: %s-24", key, data[key].Count, data[key].Date.Local().String()))
		result = append(result, data[key].(Event).GetEventPresentation())
	}
	return result
}

// sortMapByTime sorts a map of events by their timestamp in ascending or descending order based on the given order parameter.
// It takes a map of string keys to Event values and an integer order (0 for descending, 1 for ascending) as input.
// Returns a slice of event presentations sorted by timestamp according to the specified order.
func sortMapByTime(data map[string]interface{}, order int) []string {

	// result and timeStamps are built in order, all elements in them are ordered based on the criteria "order"
	result := make([]string, len(data))
	timeStamps := make([]int64, len(data))
	index := 0
	posToInsert := 0
	for key, e := range data {
		//result = append(result, fmt.Sprintf("Truck: %-9s - Alarms: %4d - Last alarm: %s-24", key, data[key].Count, data[key].Date.Local().String()))
		posToInsert = index
		index++
		v := e.(Event).GetTimeStamp().UnixMilli()
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
		result[posToInsert] = data[key].(Event).GetEventPresentation()
	}
	return result
}

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

/*
0: 4
1: 6
2: 7
3:
4:

*/
