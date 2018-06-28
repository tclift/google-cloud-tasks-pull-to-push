/*
 * Copyright © 2018 Tom Clift
 */
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tclift/google-cloud-tasks-pull-to-push/tasks"
	"log"
	"time"
)

var taskOptions tasks.Options

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.Flags().StringVarP(&taskOptions.Project, "project", "", "", "GCP project id (required).")
	rootCmd.MarkFlagRequired("project")
	rootCmd.Flags().StringVarP(&taskOptions.Location, "location", "", "us-central1",
		"Cloud Tasks location. Find with: 'gcloud alpha tasks locations list'.")
	rootCmd.Flags().StringVarP(&taskOptions.Queue, "queue", "", "pull-to-push",
		"Name of the Cloud Tasks pull queue to process.")
	rootCmd.Flags().DurationVarP(&taskOptions.Rate, "rate", "", 1*time.Second,
		"The rate at which the pull tasks are processed.")
	rootCmd.Flags().DurationVarP(&taskOptions.LeaseDuration, "lease-duration", "", 60*time.Second,
		"Time allowed to process each task.")
	rootCmd.Flags().DurationVarP(&taskOptions.PullMinBackoff, "pull-min-backoff", "", 2*time.Second,
		"Min backoff when there are no tasks to lease from the pull queue (timeout before first retry).")
	rootCmd.Flags().DurationVarP(&taskOptions.PullMaxBackoff, "pull-max-backoff", "", 30*time.Second,
		"Max backoff when there are no tasks to lease from the pull queue (max time between tasks).")
	rootCmd.Flags().IntVarP(&taskOptions.PullMaxDoublings, "pull-max-doublings", "", 4,
		"Number of times the min backoff is doubled when there are no tasks to lease from the pull queue, before the "+
			"time becomes constant, incrementing by the minimum backoff.")
	rootCmd.Flags().DurationVarP(&taskOptions.PushMinBackoff, "push-min-backoff", "", 5*time.Second,
		"Min backoff when a push task fails (timeout before first retry).")
	rootCmd.Flags().DurationVarP(&taskOptions.PushMaxBackoff, "push-max-backoff", "", 1*time.Hour,
		"Max backoff when a push task fails (max time between retries).")
	rootCmd.Flags().IntVarP(&taskOptions.PushMaxDoublings, "push-max-doublings", "", 5,
		"Number of times the min backoff is doubled when a push task is failing, before the time becomes constant, "+
			"incrementing by the minimum backoff.")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "google-cloud-tasks-pull-to-push",
	Short: "Virtual Google Cloud Tasks push queue to arbitrary URLs",
	Long:  "Leases tasks from a Google Cloud Tasks pull queue with a specific payload that includes a URL and payload, " +
		"and executes an HTTP request to the URL as a push queue would have.\n" +
		"\n" +
		"For more detail about the 'backoff' and 'doublings' options, see [GCP: Retrying Failed Push Tasks]" +
		"(https://cloud.google.com/appengine/docs/standard/go/taskqueue/push/retrying-tasks)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := tasks.Run(&taskOptions); err != nil {
			log.Fatal(err)
		}
	},
}
