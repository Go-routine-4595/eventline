package main

import (
	"Go-routine-4595/eventline"
	"context"
	"fmt"
	"github.com/icrowley/fake"
	"time"
)

type Element struct {
	timeStramp time.Time
	id         string
	company    string
	continent  string
	city       string
}

func (e Element) GetTimeStamp() time.Time {
	return e.timeStramp
}

func (e Element) GetEventPresentation() string {
	return fmt.Sprintf("Event: %-20s - Compnay: %-20s - Contient: %-20s - City: %-20s - time: %-29s",
		e.id,
		e.company,
		e.continent,
		e.city,
		e.timeStramp.Local().Format(time.RFC3339))
}

func main() {
	live()
}

func static() {
	events := make(map[string]interface{})
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
func live() {
	ctx, cancel := context.WithCancel(context.Background())
	ev := eventline.NewPresenter("my test")
	ev.WithLogFile("logs")
	ev.TimedOrdered(eventline.Desc)
	go ev.Start(cancel, ctx)
	count := 0
	events := make(map[string]interface{})

	t := time.NewTicker(time.Second)
	defer t.Stop()
loop:
	for {
		select {
		case <-t.C:
			//TODO
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

func CreateData(list map[string]interface{}) map[string]interface{} {
	id := fake.DomainName()
	list[id] = Element{
		timeStramp: time.Now(),
		id:         id,
		company:    fake.Company(),
		continent:  fake.Continent(),
		city:       fake.City(),
	}
	return list
}
