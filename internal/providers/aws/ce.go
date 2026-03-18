package aws

import (
    "context"
    "fmt"
    "strconv"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    awscfg "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/costexplorer"
    "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
    "github.com/lynqtech/batch-cost/internal/providers"
)

type ceClient struct {
    client *costexplorer.Client
}

func newCEClient(cfg awscfg.Config) *ceClient {
    return &ceClient{client: costexplorer.NewFromConfig(cfg)}
}

func (p *AWSProvider) GetRealCost(ctx context.Context, jobName string) (*providers.CostInfo, error) {
    end := time.Now().Format("2006-01-02")
    start := time.Now().AddDate(0, -1, 0).Format("2006-01-02")

    out, err := p.ce.client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
        TimePeriod:  &types.DateInterval{Start: aws.String(start), End: aws.String(end)},
        Granularity: types.GranularityMonthly,
        Metrics:     []string{"UnblendedCost"},
        Filter: &types.Expression{
            Tags: &types.TagValues{
                Key:    aws.String("BatchJob"),
                Values: []string{jobName},
            },
        },
    })
    if err != nil {
        return nil, fmt.Errorf("cost explorer: %w", err)
    }

    amount := 0.0
    unit := "USD"
    if len(out.ResultsByTime) > 0 {
        if v, ok := out.ResultsByTime[0].Total["UnblendedCost"]; ok {
            amount, _ = strconv.ParseFloat(aws.ToString(v.Amount), 64)
            unit = aws.ToString(v.Unit)
        }
    }

    return &providers.CostInfo{
        Amount:      amount,
        Unit:        unit,
        PeriodStart: start,
        PeriodEnd:   end,
    }, nil
}
