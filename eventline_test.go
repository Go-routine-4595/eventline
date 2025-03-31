package eventline

import (
	"testing"
	"time"
)

func TestShift(t *testing.T) {
	tests := []struct {
		name      string
		t         []int64
		s         []string
		start     int
		stop      int
		expectedT []int64
		expectedS []string
	}{
		{
			name:      "shitDownFromIndex single element",
			t:         []int64{10, 20, 30, 40},
			s:         []string{"a", "b", "c", "d"},
			start:     1,
			stop:      0,
			expectedT: []int64{10, 10, 30, 40},
			expectedS: []string{"a", "a", "c", "d"},
		},
		{
			name:      "shitDownFromIndex entire array",
			t:         []int64{10, 20, 30, 40},
			s:         []string{"a", "b", "c", "d"},
			start:     2,
			stop:      0,
			expectedT: []int64{10, 10, 20, 40},
			expectedS: []string{"a", "a", "b", "d"},
		},
		{
			name:      "shitDownFromIndex with single element range",
			t:         []int64{10, 20, 30, 40},
			s:         []string{"a", "b", "c", "d"},
			start:     2,
			stop:      1,
			expectedT: []int64{10, 20, 20, 40},
			expectedS: []string{"a", "b", "b", "d"},
		},
		{
			name:      "shitDownFromIndex with no effect",
			t:         []int64{10, 20, 30, 40},
			s:         []string{"a", "b", "c", "d"},
			start:     3,
			stop:      2,
			expectedT: []int64{10, 20, 30, 30},
			expectedS: []string{"a", "b", "c", "c"},
		},
		{
			name:      "shitDownFromIndex edge case at start of array",
			t:         []int64{10, 20, 30, 40},
			s:         []string{"a", "b", "c", "d"},
			start:     3,
			stop:      1,
			expectedT: []int64{10, 20, 20, 30},
			expectedS: []string{"a", "b", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shitDownFromIndex(tt.t, tt.s, tt.start, tt.stop)
			for i, v := range tt.expectedT {
				if tt.t[i] != v {
					t.Errorf("unexpected value at index %d in timestamps: expected %v, got %v", i, v, tt.t[i])
				}
			}
			for i, v := range tt.expectedS {
				if tt.s[i] != v {
					t.Errorf("unexpected value at index %d in strings: expected %v, got %v", i, v, tt.s[i])
				}
			}
		})
	}
}

type mockEvent struct {
	timestamp         time.Time
	eventPresentation string
}

func (m mockEvent) GetTimeStamp() time.Time {
	return m.timestamp
}

func (m mockEvent) GetEventPresentation() string {
	return m.eventPresentation
}

func TestSortMapByTime(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]Event
		order    int
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]Event{},
			order:    1,
			expected: []string{},
		},
		{
			name: "single element",
			input: map[string]Event{
				"event1": mockEvent{timestamp: time.UnixMilli(1000), eventPresentation: "Event 1"},
			},
			order:    1,
			expected: []string{"Event 1"},
		},
		{
			name: "multiple elements in order",
			input: map[string]Event{
				"event1": mockEvent{timestamp: time.UnixMilli(1000), eventPresentation: "Event 1"},
				"event2": mockEvent{timestamp: time.UnixMilli(2000), eventPresentation: "Event 2"},
			},
			order:    1,
			expected: []string{"Event 1", "Event 2"},
		},
		{
			name: "multiple elements out of order",
			input: map[string]Event{
				"event2": mockEvent{timestamp: time.UnixMilli(2000), eventPresentation: "Event 2"},
				"event1": mockEvent{timestamp: time.UnixMilli(1000), eventPresentation: "Event 1"},
			},
			order:    1,
			expected: []string{"Event 1", "Event 2"},
		},
		{
			name: "elements with same timestamp",
			input: map[string]Event{
				"event1": mockEvent{timestamp: time.UnixMilli(1000), eventPresentation: "Event 1"},
				"event2": mockEvent{timestamp: time.UnixMilli(1000), eventPresentation: "Event 2"},
			},
			order:    1,
			expected: []string{"Event 1", "Event 2"},
		},
		{
			name: "multiple elements with mixed timestamps",
			input: map[string]Event{
				"event3": mockEvent{timestamp: time.UnixMilli(3000), eventPresentation: "Event 3"},
				"event1": mockEvent{timestamp: time.UnixMilli(1000), eventPresentation: "Event 1"},
				"event2": mockEvent{timestamp: time.UnixMilli(2000), eventPresentation: "Event 2"},
			},
			order:    1,
			expected: []string{"Event 1", "Event 2", "Event 3"},
		},
		{
			name: "multiple elements with mixed timestamps",
			input: map[string]Event{
				"event3": mockEvent{timestamp: time.UnixMilli(3000), eventPresentation: "Event 3"},
				"event1": mockEvent{timestamp: time.UnixMilli(1000), eventPresentation: "Event 1"},
				"event2": mockEvent{timestamp: time.UnixMilli(2000), eventPresentation: "Event 2"},
				"event0": mockEvent{timestamp: time.UnixMilli(500), eventPresentation: "Event 0"},
			},
			order:    1,
			expected: []string{"Event 0", "Event 1", "Event 2", "Event 3"},
		},
		{
			name: "multiple elements with mixed timestamps",
			input: map[string]Event{
				"event3": mockEvent{timestamp: time.UnixMilli(3000), eventPresentation: "Event 3"},
				"event1": mockEvent{timestamp: time.UnixMilli(1000), eventPresentation: "Event 1"},
				"event2": mockEvent{timestamp: time.UnixMilli(2000), eventPresentation: "Event 2"},
				"event0": mockEvent{timestamp: time.UnixMilli(500), eventPresentation: "Event 0"},
			},
			order:    0,
			expected: []string{"Event 3", "Event 2", "Event 1", "Event 0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortMapByTime(tt.input, tt.order)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v elements, got %v", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("expected %q at position %d, got %q", tt.expected[i], i, result[i])
				}
			}
		})
	}
}
