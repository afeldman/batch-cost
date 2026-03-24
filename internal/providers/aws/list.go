package aws

import (
    "context"
    "fmt"
    "os"

    "github.com/aws/aws-sdk-go-v2/service/batch"
    "github.com/aws/aws-sdk-go-v2/service/batch/types"
)

// AutoDiscoverQueue ermittelt die Job-Queue.
// Priorität: --queue Flag (wird als Parameter übergeben) > JOB_QUEUE Env > erste Queue via API.
// Gibt Fehler zurück wenn >1 Queue gefunden und kein expliziter Wert.
func (p *AWSProvider) AutoDiscoverQueue(ctx context.Context) (string, error) {
    if q := os.Getenv("JOB_QUEUE"); q != "" {
        return q, nil
    }
    out, err := p.batch.DescribeJobQueues(ctx, &batch.DescribeJobQueuesInput{})
    if err != nil {
        return "", fmt.Errorf("describe-job-queues: %w", err)
    }
    if len(out.JobQueues) == 0 {
        return "", fmt.Errorf("keine Job-Queue gefunden — bitte --queue setzen")
    }
    if len(out.JobQueues) > 1 {
        return "", fmt.Errorf("%d Queues gefunden — bitte --queue explizit angeben", len(out.JobQueues))
    }
    return *out.JobQueues[0].JobQueueName, nil
}

// ListJobs gibt Job-IDs aus der Queue zurück.
// statuses: ["SUCCEEDED", "FAILED", "RUNNING"] — Default wenn leer
// Ruft batch.ListJobs pro Status auf, merged und kürzt auf limit.
func (p *AWSProvider) ListJobs(ctx context.Context, queue string, statuses []string, limit int) ([]string, error) {
    if len(statuses) == 0 {
        statuses = []string{"SUCCEEDED", "FAILED", "RUNNING"}
    }
    var ids []string
    for _, status := range statuses {
        var batchStatus types.JobStatus
        switch status {
        case "SUCCEEDED":
            batchStatus = types.JobStatusSucceeded
        case "FAILED":
            batchStatus = types.JobStatusFailed
        case "RUNNING":
            batchStatus = types.JobStatusRunning
        default:
            continue
        }
        
        out, err := p.batch.ListJobs(ctx, &batch.ListJobsInput{
            JobQueue:  &queue,
            JobStatus: batchStatus,
        })
        if err != nil {
            return nil, fmt.Errorf("list-jobs (%s): %w", status, err)
        }
        for _, j := range out.JobSummaryList {
            ids = append(ids, *j.JobId)
            if len(ids) >= limit {
                return ids, nil
            }
        }
    }
    return ids, nil
}
