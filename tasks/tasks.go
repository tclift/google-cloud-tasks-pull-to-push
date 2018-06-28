/*
 * Copyright © 2018 Tom Clift
 */
package tasks

import (
	"fmt"
	"time"
)

type Options struct {
	Project          string
	Location         string
	Queue            string
	Rate             time.Duration
	LeaseDuration    time.Duration
	PullMinBackoff   time.Duration
	PullMaxBackoff   time.Duration
	PullMaxDoublings int
	PushMinBackoff   time.Duration
	PushMaxBackoff   time.Duration
	PushMaxDoublings int
}

func Run(taskOptions *Options) error {
	fmt.Printf("%v", taskOptions)

	return nil
}
