package research

import (
	"math"
	"testing"
)

func TestProjectBlendsTowardPosition(t *testing.T) {
	p := DefaultProjection()
	position := NewShape([]WeekScore{{2025, 1, 2}, {2025, 1, 6}, {2025, 1, 10}, {2025, 1, 14}, {2025, 1, 18}}, 2025, 3, p)
	own := func(n int) Shape {
		var w []WeekScore
		for i := range n {
			w = append(w, WeekScore{2024, i + 1, 30})
		}
		return NewShape(w, 2025, 3, p)
	}
	tests := []struct {
		name         string
		player       Shape
		wantMedian   float64
		wantGames    int
		wantOK       bool
		positionOnly bool
	}{
		{"no history is the position shape", own(0), 10, 0, true, true},
		{"one game stays near the position", own(1), 10, 1, true, false},
		{"many games is mostly his own", own(40), 30, 40, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Project(tt.player, position, p)
			if ok != tt.wantOK || got.Median != tt.wantMedian || got.Games != tt.wantGames {
				t.Fatalf("got %+v ok=%v, want median %v games %d", got, ok, tt.wantMedian, tt.wantGames)
			}
			if got.Floor > got.Median || got.Median > got.Ceiling {
				t.Fatalf("range out of order: %+v", got)
			}
			if tt.positionOnly && (got.Floor != 2 || got.Ceiling != 18) {
				t.Fatalf("position-only range = %+v, want floor 2 ceiling 18", got)
			}
		})
	}
}

func TestShapeWeights(t *testing.T) {
	p := DefaultProjection()
	tests := []struct {
		name string
		week WeekScore
		want float64
	}{
		{"last week", WeekScore{2025, 9, 0}, math.Pow(0.5, 1.0/8)},
		{"eight weeks back is half", WeekScore{2025, 2, 0}, 0.5},
		{"the projected week itself is excluded", WeekScore{2025, 10, 0}, 0},
		{"a later week is excluded", WeekScore{2025, 11, 0}, 0},
		{"last season is the prior weight", WeekScore{2024, 17, 0}, 1.0 / 3},
		{"two seasons back is the prior weight", WeekScore{2023, 1, 0}, 1.0 / 3},
		{"three seasons back is dropped", WeekScore{2022, 1, 0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.weight(tt.week, 2025, 10); math.Abs(got-tt.want) > 1e-12 {
				t.Fatalf("weight = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectNothingKnown(t *testing.T) {
	if _, ok := Project(Shape{}, Shape{}, DefaultProjection()); ok {
		t.Fatal("empty shapes must not project")
	}
}
