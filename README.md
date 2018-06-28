# google-cloud-tasks-pull-to-push

Google Cloud Tasks Push queue emulation for arbitrary destination URLs (if you need it, you know).

At the time of writing, Google Cloud Tasks Push queues can only target App Engine URLs. This tool is a workaround to
allow the use of Cloud Tasks to target arbitrary URLs. A 'queue pump'. This is done by:

 * Your code enqueuing a task in a specific format to a special pull queue. The task includes the target URL details.
 * This tool polls the pull queue.
 * This tool sends the HTTP request to the target URL.
 * As per App Engine Push queue semantics, a 2xx response to the HTTP request results in task completion, and anything
   else (including request failure) results in the task staying in the queue for later retry.


## Usage

The only required option is `project` (GCP project id):

    google-cloud-tasks-pull-to-push --project my-project

See the command help for a description of the available options.

    google-cloud-tasks-pull-to-push --help

### Queue Backoff

There are two sets of queue backoff settings - one for the pull queue, and one for push (making the HTTP requests).

The pull backoff settings apply when there are no tasks in the queue. This is to prevent polling too often during
periods of inactivity, but more backoff means more latency handling tasks.

The push backoff settings apply to failed HTTP requests (non-2xx response). This is to prevent wasting resources trying
the same task when it continually fails. Note that failing tasks are never deleted by this utility; they will continue
retrying at the max push backoff rate.

The backoff parameters are modelled on the App Engine queue settings. See
[the docs there](https://cloud.google.com/appengine/docs/standard/go/taskqueue/push/retrying-tasks) for more detail.



## Building

### Binary

To build a Linux AMD64 static binary:

    ./gradlew build

Or, for a specific platform, e.g. macOS:

    ./gradlew build -PtargetPlatform=darwin-amd64

