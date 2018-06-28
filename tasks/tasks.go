/*
 * Copyright © 2018 Tom Clift
 */
package tasks

import (
	"bytes"
	"cloud.google.com/go/cloudtasks/apiv2beta2"
	"context"
	"encoding/json"
	"fmt"
	durationpb "github.com/golang/protobuf/ptypes/duration"
	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2beta2"
	"log"
	"math"
	"net/http"
	"strings"
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
	ctx := context.Background()

	client, err := createClient(ctx)
	if err != nil {
		return err
	}

	// number of attempts since there have been no tasks to process
	attempt := 0
	httpClient := &http.Client{
		Timeout: taskOptions.LeaseDuration,
	}
	queueId := fmt.Sprintf("projects/%s/locations/%s/queues/%s", taskOptions.Project, taskOptions.Location,
		taskOptions.Queue)

	fmt.Printf("%v Polling queue %v\n", logStamp(), queueId)

	for {
		task, err := leaseOne(ctx, client, queueId, taskOptions.LeaseDuration)
		if err != nil {
			return err
		}

		if task != nil {
			attempt = 0

			go func() {
				if err := handleTask(ctx, taskOptions, client, httpClient, task); err != nil {
					log.Fatal(err)
				}
			}()

			time.Sleep(taskOptions.Rate)
		} else {
			attempt++
			sleep := timeBeforeNext(attempt, taskOptions.PullMinBackoff, taskOptions.PullMaxBackoff,
				taskOptions.PullMaxDoublings)
			fmt.Printf("%v No tasks, wait %v.\n", logStamp(), sleep)
			time.Sleep(sleep)
		}
	}
}

func createClient(ctx context.Context) (*cloudtasks.Client, error) {
	return cloudtasks.NewClient(ctx)
}

func leaseOne(ctx context.Context, client *cloudtasks.Client, queueId string,
	duration time.Duration) (*taskspb.Task, error) {
	res, err := client.LeaseTasks(ctx, &taskspb.LeaseTasksRequest{
		Parent:        queueId,
		LeaseDuration: durationToPb(duration),
		MaxTasks:      1,
		ResponseView:  taskspb.Task_FULL,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Tasks) == 0 {
		return nil, nil
	}

	return res.Tasks[0], nil
}

// Handle a task by performing the request it specifies.
func handleTask(ctx context.Context, taskOptions *Options, client *cloudtasks.Client, httpClient *http.Client,
	task *taskspb.Task) error {
	taskId := task.Name[strings.LastIndex(task.Name, "/")+1 : len(task.Name)]

	fmt.Printf("%v Handling task %v (created %#v, attempt #%v)...\n", logStamp(), taskId,
		timeFromPb(task.CreateTime).Format(time.RFC3339), task.Status.AttemptDispatchCount)

	var payload pullToPushTaskPayload
	if err := json.Unmarshal(task.GetPullMessage().Payload, &payload); err != nil {
		// Fatal error. Task should be deleted.
		fmt.Printf("\t[%v] Pull task payload failed to parse.", taskId)

		return err
	}

	httpStatus, err := execPushTask(httpClient, taskId, payload.Method, payload.AbsUrl, payload.Payload, payload.Headers)
	if err != nil {
		return err
	}

	if httpStatus >= 200 && httpStatus <= 299 {
		taskCompleted(ctx, client, task, taskId)
	} else {
		taskFailed(ctx, taskOptions, client, task, taskId)
	}

	return nil
}

func execPushTask(httpClient *http.Client, taskId string, method string, url string, body string,
	headers map[string]string) (int, error) {
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		fmt.Printf("\t[%v] Push task payload failed to parse.\n", taskId)

		// treat this as a fatal error (task should be deleted)
		return 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	fmt.Printf("\t[%v] > %v %v %v...\n", taskId, req.Method, req.URL, req.Header)

	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("\t[%v] Request failed: %v\n", taskId, err)

		// treat this as a task failure (non-fatal)
		return 0, nil
	}
	defer res.Body.Close()

	fmt.Printf("\t[%v] < %v\n", taskId, res.StatusCode)

	return res.StatusCode, nil
}

// Handle push task success by acknowledging the task (removing it from the queue).
func taskCompleted(ctx context.Context, client *cloudtasks.Client, task *taskspb.Task, taskId string) error {
	err := ackTask(ctx, client, task)
	if err != nil {
		fmt.Printf("\t[%v] Failed to acknowledge task.\n", taskId)

		return err
	}

	fmt.Printf("\t[%v] Task acknowledged.\n", taskId)

	return nil
}

// Handle push task failure by extending the task lease (increasing the time before it comes back).
func taskFailed(ctx context.Context, taskOptions *Options, client *cloudtasks.Client, task *taskspb.Task,
	taskId string) error {
	retry := timeBeforeNext(int(task.Status.AttemptDispatchCount+1), taskOptions.PushMinBackoff,
		taskOptions.PushMaxBackoff, taskOptions.PushMaxDoublings)

	err := renewLease(ctx, client, task, retry)
	if err != nil {
		fmt.Printf("\t[%v] Failed to renew lease.\n", taskId)

		return err
	}

	fmt.Printf("\t[%v] Renewed lease for %v.\n", taskId, retry)

	return nil
}

func ackTask(ctx context.Context, client *cloudtasks.Client, task *taskspb.Task) error {
	return client.AcknowledgeTask(ctx, &taskspb.AcknowledgeTaskRequest{
		Name:         task.Name,
		ScheduleTime: task.ScheduleTime,
	})
}

func renewLease(ctx context.Context, client *cloudtasks.Client, task *taskspb.Task,
	leaseDuration time.Duration) error {
	_, err := client.RenewLease(ctx, &taskspb.RenewLeaseRequest{
		Name:          task.Name,
		ScheduleTime:  task.ScheduleTime,
		LeaseDuration: durationToPb(leaseDuration),
	})

	return err
}

// Time to wait before the next attempt. I.e. when attempt = 1, the time to wait before attempt 1 (the min backoff).
func timeBeforeNext(attempt int, minBackoff time.Duration, maxBackoff time.Duration, maxDoublings int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	t := time.Duration(0)
	// ref: https://cloud.google.com/appengine/docs/standard/go/taskqueue/push/retrying-tasks
	if attempt > maxDoublings {
		t = time.Duration(attempt-maxDoublings) * (minBackoff * time.Duration(math.Pow(2, float64(maxDoublings))))
	} else {
		t = time.Duration(math.Pow(2, float64(attempt-1))) * minBackoff
	}

	if t > maxBackoff {
		return maxBackoff
	} else {
		return t
	}
}

func logStamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func durationToPb(duration time.Duration) *durationpb.Duration {
	seconds := int64(duration.Round(time.Second).Seconds())
	nanos := int32((duration - (time.Duration(seconds) * time.Second)).Nanoseconds())
	pb := durationpb.Duration{Seconds: seconds, Nanos: nanos}

	return &pb
}

func timeFromPb(t *timestamppb.Timestamp) time.Time {
	return time.Unix(t.Seconds, int64(t.Nanos)).UTC()
}

type pullToPushTaskPayload struct {
	Method  string            `json:"method"`
	AbsUrl  string            `json:"absUrl"`
	Headers map[string]string `json:"headers,omitempty"`
	Payload string            `json:"payload,omitempty"`
}
