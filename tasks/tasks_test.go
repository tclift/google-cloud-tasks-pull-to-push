/*
 * Copyright © 2018 Tom Clift
 */
package tasks

import (
	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
	"testing"
	"time"
)

func TestTimeBeforeNext(t *testing.T) {
	tables := []struct {
		attempt int
		minBackoff time.Duration
		maxBackoff time.Duration
		maxDoublings int
		expected time.Duration
	}{
		{0, 5*time.Second, 10*time.Minute, 5, 0},
		{1, 5*time.Second, 10*time.Minute, 5, 5*time.Second},
		{2, 5*time.Second, 10*time.Minute, 5, 10*time.Second},
		{3, 5*time.Second, 10*time.Minute, 5, 20*time.Second},
		{4, 5*time.Second, 10*time.Minute, 5, 40*time.Second},
		{5, 5*time.Second, 10*time.Minute, 5, 80*time.Second},
		{6, 5*time.Second, 10*time.Minute, 5, 160*time.Second},
		// constant hit, but it's still double
		{7, 5*time.Second, 10*time.Minute, 5, 320*time.Second},
		// first time we didn't double
		{8, 5*time.Second, 10*time.Minute, 5, 480*time.Second},
		// max hit
		{9, 5*time.Second, 10*time.Minute, 5, 600*time.Second},
	}

	for _, table := range tables {
		result := timeBeforeNext(table.attempt, table.minBackoff, table.maxBackoff, table.maxDoublings)
		if result != table.expected {
			t.Errorf("timeBeforeNext(%v, %v, %v, %v) incorrect, got: %v, want: %v", table.attempt, table.minBackoff,
				table.maxBackoff, table.maxDoublings, result, table.expected)
		}
	}
}

func TestDurationToPb(t *testing.T) {
	pb := durationToPb(1*time.Second + 100*time.Nanosecond)
	if pb.Seconds != 1 || pb.Nanos != 100 {
		t.Errorf("%v", pb)
	}
}

func TestTimeFromPb(t *testing.T) {
	got := timeFromPb(&timestamppb.Timestamp{
		Seconds: 1136214245,
		Nanos:   999999999,
	})
	want, _ := time.Parse(time.RFC3339Nano, "2006-01-02T15:04:05.999999999Z")
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
