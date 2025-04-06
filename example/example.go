package main

import (
	"Go-routine-4595/eventline"
	"context"
	"fmt"
	"github.com/icrowley/fake"
	"slices"
	"strconv"
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
		events, _ = CreateData(events)
		res, _ := eventline.DebugSortMapByTime(events, eventline.Asc)
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
	var lastID string

	ctx, cancel := context.WithCancel(context.Background())
	ev := eventline.NewPresenter[T]("my test")
	ev.WithLogFile("logs")
	ev.TimedOrdered(eventline.Aphabetical)
	go ev.Start(cancel, ctx)
	count := 0
	events := make(map[string]T)

	t := time.NewTicker(time.Second)
	defer t.Stop()
loop:
	for {
		select {
		case <-t.C:
			if count <= 40 {
				count++
				//events, lastID = CreateData(events)
				events, lastID = CreateDataWithIDasNumber(events, count)
				pre := eventline.NewEventHandler(count, events, lastID)
				ev.Send(pre)
			}
		case <-ctx.Done():
			ev.Stop()
			break loop
		default:

		}
	}
	ev.Stop()
}

func CreateData[T eventline.Event](list map[string]T) (map[string]T, string) {
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
	return list, id
}

func CreateDataWithIDasNumber[T eventline.Event](list map[string]T, id int) (map[string]T, string) {
	list = AddOrReplace(list, 10)
	idString := strconv.Itoa(id)
	element := Element{
		TimeStamp: time.Now(),
		id:        idString,
		company:   fake.Company(),
		continent: fake.Continent(),
		city:      fake.City(),
	}
	// Convert Element to T using type assertion on an interface
	var event T = any(element).(T)
	list[idString] = event
	return list, idString
}

func AddOrReplace[T eventline.Event](list map[string]T, limit int) map[string]T {
	if len(list) >= limit {
		keyList := getKeys(list)
		slices.SortFunc(keyList, func(a, b string) int {
			ai, _ := strconv.Atoi(a)
			bi, _ := strconv.Atoi(b)
			if ai > bi {
				return -1
			}
			if ai < bi {
				return 1
			}
			return 0
		})
		delete(list, keyList[len(keyList)-1])
	}

	return list
}

func getKeys[T eventline.Event](list map[string]T) []string {
	keys := make([]string, 0, len(list))
	for key := range list {
		keys = append(keys, key)
	}
	return keys
}
