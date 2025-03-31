package main

import (
	"Go-routine-4595/eventline"
	"context"
	"fmt"
	"github.com/icrowley/fake"
	"time"
)

type Element struct {
	TimeStamp time.Time
	id        string
	company   string
	continent string
	city      string
}

func (e Element) GetTimeStamp() time.Time {
	return e.TimeStamp
}

func (e Element) GetEventPresentation() string {
	return fmt.Sprintf("Event: %-20s - Compnay: %-20s - Contient: %-20s - City: %-20s - time: %-29s",
		e.id,
		e.company,
		e.continent,
		e.city,
		e.TimeStamp.Local().Format(time.RFC3339))
}

func main() {
	live[Element]()
}

func static[T eventline.Event]() {
	events := make(map[string]T)
	for i := 0; i < 15; i++ {
		events = CreateData(events)
		res := eventline.DebugSortMapByTime(events, eventline.Asc)
		fmt.Println("-------------------------------------------------------------")
		fmt.Println("Count: ", i)
		for _, element := range res {
			fmt.Printf("%s\n", element)
		}
		fmt.Println()
		time.Sleep(time.Second)
	}
}

func live[T eventline.Event]() {
	ctx, cancel := context.WithCancel(context.Background())
	ev := eventline.NewPresenter[T]("my test")
	ev.WithLogFile("logs")
	ev.TimedOrdered(eventline.Desc)
	go ev.Start(cancel, ctx)
	count := 0
	events := make(map[string]T)

	t := time.NewTicker(time.Second)
	defer t.Stop()
loop:
	for {
		select {
		case <-t.C:
			if count == 40 {
				cancel()
				break loop
			}
			count++
			events = CreateData(events)
			pre := eventline.NewEventHandler(count, events)
			ev.Send(pre)
		}
	}
}

func CreateData[T eventline.Event](list map[string]T) map[string]T {
	id := fake.DomainName()
	element := Element{
		TimeStamp: time.Now(),
		id:        id,
		company:   fake.Company(),
		continent: fake.Continent(),
		city:      fake.City(),
	}
	// Convert Element to T using type assertion on an interface
	var event T = any(element).(T)
	list[id] = event
	return list
}
