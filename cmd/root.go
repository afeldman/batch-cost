package cmd

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "strings"

    "github.com/charmbracelet/huh"
    "github.com/afeldman/batch-cost/internal/llm"
    "github.com/afeldman/batch-cost/internal/pricing"
    "github.com/afeldman/batch-cost/internal/providers"
    awsprovider "github.com/afeldman/batch-cost/internal/providers/aws"
    "github.com/afeldman/batch-cost/internal/ui"
    "github.com/spf13/cobra"
)

var (
    flagRegion  string
    flagProfile string
    flagJobID   string
    flagJobName string
    flagJSON    bool
    flagConfig  string
    flagLast    int
    flagQueue   string
    flagStatus  string  // komma-separiert, default ""
    flagLastStr string  // für interactive mode
    flagAnalyze bool
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
    rootCmd.Flags().IntVar(&flagLast,   "last",   0,  "Letzte N Jobs analysieren")
    rootCmd.Flags().StringVar(&flagQueue, "queue", "", "Job-Queue (auto-discovery wenn leer)")
    rootCmd.Flags().StringVar(&flagStatus, "status", "", "Status-Filter: SUCCEEDED,FAILED,RUNNING")
    rootCmd.Flags().BoolVar(&flagAnalyze, "analyze", false, "LLM-Analyse aktivieren")
}

func run(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Interaktiver Modus wenn keine Flags gesetzt
    if flagJobID == "" && flagJobName == "" && flagLast == 0 {
        if err := interactiveSelect(); err != nil {
            return err
        }
        // Konvertiere flagLastStr zu flagLast für interaktiven Modus
        if flagLastStr != "" {
            if n, err := strconv.Atoi(flagLastStr); err == nil && n > 0 {
                flagLast = n
            }
        }
    }

    // 1. Pricing laden
    baseCfg, opts, llmCfg, err := pricing.LoadOptions(flagConfig)
    if err != nil {
        return fmt.Errorf("pricing config: %w", err)
    }

    // 2. Preise auflösen (mit Spinner falls API-Call nötig)
    var pricingCfg pricing.Config
    raw, err := ui.RunWithSpinner("Updating prices...", func() (interface{}, error) {
        return pricing.ResolvePrices(ctx, baseCfg, opts, flagRegion)
    })
    if err != nil {
        return fmt.Errorf("resolve prices: %w", err)
    }
    pricingCfg = raw.(pricing.Config)

    // 3. Lokalen LLM-Server verbinden oder starten (als Daemon)
    if llmCfg.Local.Enabled {
        mgr := llm.NewManager(llmCfg.Local)

        if mgr.IsHealthy() {
            // Server läuft bereits — sofort nutzen, nicht beenden
            llmCfg.Endpoint = fmt.Sprintf("http://localhost:%d/v1", llmCfg.Local.Port)
            llmCfg.Model = llmCfg.Local.ModelRepo
        } else {
            // Server starten (Daemon — bleibt nach batch-cost laufen)
            _, err := ui.RunWithSpinner("LLM-Server starten (einmalig ~60s)...", func() (interface{}, error) {
                return nil, mgr.StartDaemon(ctx)
            })
            if err != nil {
                ui.Warn("LLM nicht verfügbar: " + err.Error())
                llmCfg.Enabled = false
            } else {
                llmCfg.Endpoint = fmt.Sprintf("http://localhost:%d/v1", llmCfg.Local.Port)
                llmCfg.Model = llmCfg.Local.ModelRepo
            }
        }
    }

    // 4. Provider
    provider, err := awsprovider.New(flagRegion, flagProfile)
    if err != nil {
        return fmt.Errorf("provider: %w", err)
    }

    // 5. Modi
    if flagLast > 0 {
        // Multi-Job-Modus
        return runMultiJobMode(ctx, provider, pricingCfg, llmCfg)
    } else if flagJobID != "" {
        // Einzel-Job
        return runSingleJobMode(ctx, provider, pricingCfg, llmCfg)
    } else if flagJobName != "" {
        // Cost Explorer
        return runCostExplorerMode(ctx, provider)
    } else {
        // Sollte nicht passieren (interactiveSelect setzt Flags)
        return fmt.Errorf("kein Modus ausgewählt")
    }
}

func runMultiJobMode(ctx context.Context, provider providers.Provider, pricingCfg pricing.Config, llmCfg llm.Config) error {
    // Queue ermitteln
    queue := flagQueue
    if queue == "" {
        if q := os.Getenv("JOB_QUEUE"); q != "" {
            queue = q
        } else {
            var err error
            queue, err = provider.AutoDiscoverQueue(ctx)
            if err != nil {
                return err
            }
        }
    }

    // Statuses
    statuses := []string{}
    if flagStatus != "" {
        statuses = strings.Split(flagStatus, ",")
    }

    // Jobs laden
    jobIDs, err := provider.ListJobs(ctx, queue, statuses, flagLast)
    if err != nil {
        return fmt.Errorf("list jobs: %w", err)
    }

    // Kosten berechnen
    var totalCost, maxCost, totalCPUPct, totalMemPct, totalSpot float64
    count := 0
    for _, id := range jobIDs {
        job, err := provider.DescribeJob(ctx, id)
        if err != nil {
            // Einzelne Jobs können fehlschlagen, aber wir machen weiter
            continue
        }
        result := pricingCfg.Calculate(job.DurationSec, job.VCPU, job.MemoryMB)
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
        return fmt.Errorf("keine Jobs gefunden oder analysiert")
    }

    multiResult := ui.MultiJobResult{
        Queue:      queue,
        Limit:      flagLast,
        Count:      count,
        TotalCost:  totalCost,
        AvgCost:    totalCost / float64(count),
        MaxCost:    maxCost,
        SpotTotal:  totalSpot,
        AvgCPUPct:  totalCPUPct / float64(count),
        AvgMemPct:  totalMemPct / float64(count),
    }
    rec := pricing.RecommendMulti(multiResult.AvgCPUPct, multiResult.AvgMemPct)

    if flagJSON {
        return printJSON(ui.MultiJobJSON{
            Queue:     multiResult.Queue,
            Count:     multiResult.Count,
            Total:     multiResult.TotalCost,
            Average:   multiResult.AvgCost,
            Max:       multiResult.MaxCost,
            SpotTotal: multiResult.SpotTotal,
            AvgCPUPct: multiResult.AvgCPUPct,
            AvgMemPct: multiResult.AvgMemPct,
        })
    }
    ui.PrintMultiJob(multiResult, rec)
    
    // LLM-Analyse für Multi-Job-Modus
    if llmCfg.Enabled || flagAnalyze {
        llmCfg.Enabled = true
        llmClient := llm.New(llmCfg)
        prompt := llm.MultiJobPrompt(queue, count, totalCost, totalCost/float64(count), 
            maxCost, totalSpot, totalCPUPct/float64(count), totalMemPct/float64(count), rec)
        
        raw, err := ui.RunWithSpinner("LLM analysiert...", func() (interface{}, error) {
            return llmClient.Analyze(ctx, prompt)
        })
        if err != nil {
            ui.Warn("LLM nicht verfügbar: " + err.Error())
        } else if analysis := raw.(string); analysis != "" {
            ui.PrintLLMAnalysis(analysis)
        }
    }
    
    return nil
}

func runSingleJobMode(ctx context.Context, provider providers.Provider, pricingCfg pricing.Config, llmCfg llm.Config) error {
    raw, err := ui.RunWithSpinner("Fetching job from AWS Batch...", func() (interface{}, error) {
        return provider.DescribeJob(ctx, flagJobID)
    })
    if err != nil {
        return fmt.Errorf("describe job: %w", err)
    }
    job := raw.(*providers.JobInfo)
    costResult := pricingCfg.Calculate(job.DurationSec, job.VCPU, job.MemoryMB)
    rec := pricing.Recommend(job.VCPU, job.MemoryMB, costResult.CPUPct, costResult.MemPct)

    if flagJSON {
        return printJSON(map[string]interface{}{
            "job":  job,
            "cost": costResult,
            "rec":  rec,
        })
    }
    ui.PrintEstimate(job, pricingCfg, costResult, rec, provider.Name())
    
    // LLM-Analyse für Single-Job-Modus
    if llmCfg.Enabled || flagAnalyze {
        llmCfg.Enabled = true
        llmClient := llm.New(llmCfg)
        prompt := llm.SingleJobPrompt(job.JobName, job.Status, job.DurationSec,
            job.VCPU, job.MemoryMB, costResult.Total, costResult.SpotTotal,
            costResult.CPUPct, costResult.MemPct, rec.Reason)
        
        raw, err := ui.RunWithSpinner("LLM analysiert...", func() (interface{}, error) {
            return llmClient.Analyze(ctx, prompt)
        })
        if err != nil {
            ui.Warn("LLM nicht verfügbar: " + err.Error())
        } else if analysis := raw.(string); analysis != "" {
            ui.PrintLLMAnalysis(analysis)
        }
    }
    
    return nil
}

func runCostExplorerMode(ctx context.Context, provider providers.Provider) error {
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

func interactiveSelect() error {
    var mode string
    err := huh.NewSelect[string]().
        Title("batch-cost — Was möchtest du tun?").
        Options(
            huh.NewOption("Kosten schätzen (Job ID)", "estimate"),
            huh.NewOption("Echte Kosten (Cost Explorer)", "explorer"),
            huh.NewOption("Mehrere Jobs analysieren", "multi"),
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
    case "multi":
        return huh.NewForm(
            huh.NewGroup(
                huh.NewInput().
                    Title("Job-Queue (leer = auto-discovery)").
                    Value(&flagQueue),
                huh.NewInput().
                    Title("Anzahl Jobs").
                    Validate(func(s string) error {
                        if s == "" {
                            return fmt.Errorf("Bitte eine Zahl eingeben")
                        }
                        return nil
                    }).
                    Value(&flagLastStr),
                huh.NewInput().
                    Title("Status-Filter (komma-separiert, z.B. SUCCEEDED,FAILED)").
                    Value(&flagStatus),
            ),
        ).Run()
    }
    return nil
}

func printJSON(v interface{}) error {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(v)
}
