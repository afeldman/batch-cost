package cmd

import (
	"context"
	"encoding/json"

	"github.com/afeldman/batch-cost/internal/pricing"
	awsprovider "github.com/afeldman/batch-cost/internal/providers/aws"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Startet batch-cost als MCP-Server",
	RunE:  runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	s := server.NewMCPServer("batch-cost", "1.0.0")

	// Tool 1: Job-Kosten schätzen
	s.AddTool(mcp.NewTool("get_job_cost",
		mcp.WithDescription("Berechnet die geschätzten Kosten für einen AWS Batch Job"),
		mcp.WithString("job_id", mcp.Required(), mcp.Description("AWS Batch Job ID")),
		mcp.WithString("region", mcp.Description("AWS Region (default: eu-central-1)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		jobID := args["job_id"].(string)
		region := "eu-central-1"
		if r, ok := args["region"].(string); ok && r != "" {
			region = r
		}
		provider, err := awsprovider.New(region, "")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		job, err := provider.DescribeJob(ctx, jobID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		cfg, opts, _, _ := pricing.LoadOptions("")
		cfg, _ = pricing.ResolvePrices(ctx, cfg, opts, region)
		result := cfg.Calculate(job.DurationSec, job.VCPU, job.MemoryMB)

		out, _ := json.Marshal(map[string]interface{}{
			"job_id":       job.JobID,
			"job_name":     job.JobName,
			"status":       job.Status,
			"duration_sec": job.DurationSec,
			"vcpu":         job.VCPU,
			"memory_mb":    job.MemoryMB,
			"total_cost":   result.Total,
			"spot_cost":    result.SpotTotal,
			"cpu_pct":      result.CPUPct,
			"mem_pct":      result.MemPct,
		})
		return mcp.NewToolResultText(string(out)), nil
	})

	// Tool 2: Multi-Job-Analyse
	s.AddTool(mcp.NewTool("analyze_jobs",
		mcp.WithDescription("Analysiert die Kosten der letzten N Jobs einer Queue"),
		mcp.WithNumber("limit", mcp.Required(), mcp.Description("Anzahl Jobs")),
		mcp.WithString("queue", mcp.Description("Job-Queue (auto-discovery wenn leer)")),
		mcp.WithString("region", mcp.Description("AWS Region")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		limit := int(args["limit"].(float64))
		region := "eu-central-1"
		if r, ok := args["region"].(string); ok && r != "" {
			region = r
		}
		queue := ""
		if q, ok := args["queue"].(string); ok {
			queue = q
		}

		provider, err := awsprovider.New(region, "")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Queue ermitteln
		if queue == "" {
			var err error
			queue, err = provider.AutoDiscoverQueue(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
		}

		// Jobs laden
		jobIDs, err := provider.ListJobs(ctx, queue, []string{}, limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Preise laden
		cfg, opts, _, _ := pricing.LoadOptions("")
		cfg, _ = pricing.ResolvePrices(ctx, cfg, opts, region)

		// Kosten berechnen
		var totalCost, maxCost, totalCPUPct, totalMemPct, totalSpot float64
		count := 0
		for _, id := range jobIDs {
			job, err := provider.DescribeJob(ctx, id)
			if err != nil {
				continue
			}
			result := cfg.Calculate(job.DurationSec, job.VCPU, job.MemoryMB)
			totalCost += result.Total
			totalSpot += result.SpotTotal
			if result.Total > maxCost {
				maxCost = result.Total
			}
			totalCPUPct += result.CPUPct
			totalMemPct += result.MemPct
			count++
		}

		if count == 0 {
			return mcp.NewToolResultError("keine Jobs gefunden oder analysiert"), nil
		}

		avgCost := totalCost / float64(count)
		avgCPUPct := totalCPUPct / float64(count)
		avgMemPct := totalMemPct / float64(count)

		out, _ := json.Marshal(map[string]interface{}{
			"queue":       queue,
			"count":       count,
			"total_cost":  totalCost,
			"avg_cost":    avgCost,
			"max_cost":    maxCost,
			"spot_total":  totalSpot,
			"avg_cpu_pct": avgCPUPct,
			"avg_mem_pct": avgMemPct,
		})
		return mcp.NewToolResultText(string(out)), nil
	})

	// MCP über stdio starten (Standard für MCP-Server)
	return server.ServeStdio(s)
}
