package aws

import (
    "context"
    "fmt"
    "strconv"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/batch"
    "github.com/aws/aws-sdk-go-v2/service/batch/types"
    "github.com/afeldman/batch-cost/internal/providers"
)

type AWSProvider struct {
    region  string
    profile string
    batch   *batch.Client
    ce      *ceClient
}

func New(region, profile string) (*AWSProvider, error) {
    opts := []func(*config.LoadOptions) error{
        config.WithRegion(region),
    }
    if profile != "" {
        opts = append(opts, config.WithSharedConfigProfile(profile))
    }

    cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
    if err != nil {
        return nil, fmt.Errorf("aws config: %w", err)
    }

    return &AWSProvider{
        region:  region,
        profile: profile,
        batch:   batch.NewFromConfig(cfg),
        ce:      newCEClient(cfg),
    }, nil
}

func (p *AWSProvider) Name() string { return "aws" }

func (p *AWSProvider) DescribeJob(ctx context.Context, jobID string) (*providers.JobInfo, error) {
    out, err := p.batch.DescribeJobs(ctx, &batch.DescribeJobsInput{
        Jobs: []string{jobID},
    })
    if err != nil {
        return nil, fmt.Errorf("describe-jobs: %w", err)
    }
    if len(out.Jobs) == 0 {
        return nil, fmt.Errorf("job nicht gefunden: %s", jobID)
    }

    job := out.Jobs[0]
    info := &providers.JobInfo{
        JobID:   aws.ToString(job.JobId),
        JobName: aws.ToString(job.JobName),
        Status:  string(job.Status),
    }

    if job.StartedAt == nil {
        return nil, fmt.Errorf("job hat noch keine Startzeit (Status: %s)", job.Status)
    }

    startMs := *job.StartedAt
    var endMs int64
    if job.Status == types.JobStatusRunning {
        info.InProgress = true
        endMs = time.Now().UnixMilli()
    } else if job.StoppedAt != nil {
        endMs = *job.StoppedAt
    } else {
        endMs = time.Now().UnixMilli()
    }

    info.DurationSec = (endMs - startMs) / 1000

    // vCPU + Memory aus resourceRequirements
    for _, req := range job.Container.ResourceRequirements {
        switch req.Type {
        case types.ResourceTypeVcpu:
            info.VCPU, _ = strconv.ParseFloat(aws.ToString(req.Value), 64)
        case types.ResourceTypeMemory:
            mem, _ := strconv.ParseInt(aws.ToString(req.Value), 10, 64)
            info.MemoryMB = mem
        }
    }
    // Legacy-Fallback
    if info.VCPU == 0 && job.Container.Vcpus != nil {
        info.VCPU = float64(aws.ToInt32(job.Container.Vcpus))
    }
    if info.MemoryMB == 0 && job.Container.Memory != nil {
        info.MemoryMB = int64(aws.ToInt32(job.Container.Memory))
    }
    if info.VCPU == 0    { info.VCPU = 1 }
    if info.MemoryMB == 0 { info.MemoryMB = 2048 }

    return info, nil
}
