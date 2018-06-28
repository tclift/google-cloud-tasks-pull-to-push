# google-cloud-tasks-pull-to-push

Google Cloud Tasks Push queue emulation for arbitrary destination URLs (if you need it, you know).

At the time of writing, Google Cloud Tasks Push queues can only target App Engine URLs. This tool is a workaround to
allow the use of Cloud Tasks to target arbitrary URLs. A 'queue pump'. This is done by:

 * Your code enqueuing a task in a specific format to a special pull queue. The task includes the target URL details.
 * This tool polls the pull queue.
 * This tool sends the HTTP request to the target URL.
 * As per App Engine Push queue semantics, a 2xx response to the HTTP request results in task completion, and anything
   else (including request failure) results in the task staying in the queue for later retry.
