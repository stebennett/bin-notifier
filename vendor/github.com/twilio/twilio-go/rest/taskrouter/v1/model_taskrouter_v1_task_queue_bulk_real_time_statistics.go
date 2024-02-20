/*
 * This code was generated by
 * ___ _ _ _ _ _    _ ____    ____ ____ _    ____ ____ _  _ ____ ____ ____ ___ __   __
 *  |  | | | | |    | |  | __ |  | |__| | __ | __ |___ |\ | |___ |__/ |__|  | |  | |__/
 *  |  |_|_| | |___ | |__|    |__| |  | |    |__] |___ | \| |___ |  \ |  |  | |__| |  \
 *
 * Twilio - Taskrouter
 * This is the public Twilio REST API.
 *
 * NOTE: This class is auto generated by OpenAPI Generator.
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

package openapi

// TaskrouterV1TaskQueueBulkRealTimeStatistics struct for TaskrouterV1TaskQueueBulkRealTimeStatistics
type TaskrouterV1TaskQueueBulkRealTimeStatistics struct {
	// The SID of the [Account](https://www.twilio.com/docs/iam/api/account) that created the TaskQueue resource.
	AccountSid *string `json:"account_sid,omitempty"`
	// The SID of the Workspace that contains the TaskQueue.
	WorkspaceSid *string `json:"workspace_sid,omitempty"`
	// The TaskQueue RealTime Statistics for each requested TaskQueue SID, represented as an array of TaskQueue results corresponding to the requested TaskQueue SIDs, each result contains the following attributes: task_queue_sid: The SID of the TaskQueue from which these statistics were calculated, total_available_workers: The total number of Workers available for Tasks in the TaskQueue, total_eligible_workers: The total number of Workers eligible for Tasks in the TaskQueue, independent of their Activity state, total_tasks: The total number of Tasks, longest_task_waiting_age: The age of the longest waiting Task, longest_task_waiting_sid: The SID of the longest waiting Task, tasks_by_status: The number of Tasks by their current status, tasks_by_priority: The number of Tasks by priority, activity_statistics: The number of current Workers by Activity.
	TaskQueueData *[]interface{} `json:"task_queue_data,omitempty"`
	// The number of TaskQueue statistics received in task_queue_data.
	TaskQueueResponseCount *int `json:"task_queue_response_count,omitempty"`
	// The absolute URL of the TaskQueue statistics resource.
	Url *string `json:"url,omitempty"`
}