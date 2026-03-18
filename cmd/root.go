package cmd

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "github.com/charmbracelet/huh"
    "github.com/lynqtech/batch-cost/internal/pricing"
    "github.com/lynqtech/batch-cost/internal/providers"
    awsprovider "github.com/lynqtech/batch-cost/internal/providers/aws"
    "github.com/lynqtech/batch-cost/internal/ui"
    "github.com/spf13/cobra"
)

var (
    flagRegion  string
    flagProfile string
    flagJobID   string
    flagJobName string
    flagJSON    bool
    flagConfig  string
)

var rootCmd = &cobra.Command{
    Use:   "batch-cost",
    Short: "AWS Batch Job Kostenanalyse",
    Long:  "Schätzt oder liest echte Kosten für AWS Batch Jobs.",
    RunE:  run,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    rootCmd.Flags().StringVar(&flagRegion,   "region",   "eu-central-1", "AWS Region")
    rootCmd.Flags().StringVar(&flagProfile,  "profile",  "",             "AWS Profil")
    rootCmd.Flags().StringVar(&flagJobID,    "job-id",   "",             "Batch Job ID (Mode: estimate)")
    rootCmd.Flags().StringVar(&flagJobName,  "job-name", "",             "Job Name für Cost Explorer")
    rootCmd.Flags().BoolVar(&flagJSON,      "json",   false, "JSON Output")
    rootCmd.Flags().StringVar(&flagConfig, "config", "",    "Pfad zu pricing.toml (auto-discover wenn leer)")
}

func run(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Interaktiver Modus wenn keine Flags gesetzt
    if flagJobID == "" && flagJobName == "" {
        if err := interactiveSelect(); err != nil {
            return err
        }
    }

    provider, err := awsprovider.New(flagRegion, flagProfile)
    if err != nil {
        return fmt.Errorf("provider: %w", err)
    }

    pricingCfg, err := pricing.LoadConfig(flagConfig)
    if err != nil {
        return fmt.Errorf("pricing config: %w", err)
    }

    if flagJobID != "" {
        raw, err := ui.RunWithSpinner("Fetching job from AWS Batch...", func() (interface{}, error) {
            return provider.DescribeJob(ctx, flagJobID)
        })
        if err != nil {
            return fmt.Errorf("describe job: %w", err)
        }
        job := raw.(*providers.JobInfo)
        costResult := pricingCfg.Calculate(job.DurationSec, job.VCPU, job.MemoryMB)

        if flagJSON {
            return printJSON(map[string]interface{}{
                "job":  job,
                "cost": costResult,
            })
        }
        ui.PrintEstimate(job, costResult, provider.Name())
        return nil
    }

    if flagJobName != "" {
        raw, err := ui.RunWithSpinner("Querying Cost Explorer...", func() (interface{}, error) {
            return provider.GetRealCost(ctx, flagJobName)
        })
        if err != nil {
            return fmt.Errorf("cost explorer: %w", err)
        }
        cost := raw.(*providers.CostInfo)

        if flagJSON {
            return printJSON(cost)
        }
        ui.PrintCostExplorer(cost, flagJobName)
        return nil
    }

    return nil
}

func interactiveSelect() error {
    var mode string
    err := huh.NewSelect[string]().
        Title("batch-cost — Was möchtest du tun?").
        Options(
            huh.NewOption("Kosten schätzen (Job ID)", "estimate"),
            huh.NewOption("Echte Kosten (Cost Explorer)", "explorer"),
        ).
        Value(&mode).
        Run()
    if err != nil {
        return err
    }

    switch mode {
    case "estimate":
        return huh.NewInput().
            Title("AWS Batch Job ID").
            Placeholder("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
            Value(&flagJobID).
            Run()
    case "explorer":
        return huh.NewInput().
            Title("Job Name (Tag: BatchJob=<name>)").
            Value(&flagJobName).
            Run()
    }
    return nil
}

func printJSON(v interface{}) error {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(v)
}
