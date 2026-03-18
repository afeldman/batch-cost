package providers

import "context"

// JobInfo enthält normalisierte Job-Metadaten (provider-agnostisch)
type JobInfo struct {
    JobID       string
    JobName     string
    Status      string
    InProgress  bool
    DurationSec int64
    VCPU        float64
    MemoryMB    int64
}

// CostInfo enthält echte Kosten aus der Billing API
type CostInfo struct {
    Amount      float64
    Unit        string
    PeriodStart string
    PeriodEnd   string
}

// Provider definiert das Interface für Cloud-Provider
type Provider interface {
    DescribeJob(ctx context.Context, jobID string) (*JobInfo, error)
    GetRealCost(ctx context.Context, jobName string) (*CostInfo, error)
    Name() string
}
