package ui

import (
    "fmt"

    "github.com/charmbracelet/lipgloss"
    "github.com/afeldman/batch-cost/internal/pricing"
    "github.com/afeldman/batch-cost/internal/providers"
)

var (
    styleHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
    styleLabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
    styleCost      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
    styleWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
    styleSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
    styleValue     = lipgloss.NewStyle().Bold(true)
    styleSpot      = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
    styleRec       = lipgloss.NewStyle().Foreground(lipgloss.Color("159"))
)

func Header(text string) {
    fmt.Println(styleHeader.Render("=== " + text + " ==="))
}

func Label(key, val string) {
    fmt.Printf("%s %s\n",
        styleLabel.Render(fmt.Sprintf("%-20s", key+":")),
        styleValue.Render(val))
}

func CostLine(key string, amount float64) {
    fmt.Printf("%s %s\n",
        styleLabel.Render(fmt.Sprintf("%-20s", key+":")),
        styleCost.Render(fmt.Sprintf("$%.4f", amount)))
}

func Warn(msg string) {
    fmt.Println(styleWarn.Render("⚠  " + msg))
}

func Separator() {
    fmt.Println(styleSeparator.Render("──────────────────────────────────────────"))
}

func PrintEstimate(job *providers.JobInfo, cfg pricing.Config, result pricing.Result, rec pricing.Recommendation, provider string) {
    fmt.Println()
    Header("Batch Job Cost Report")
    fmt.Println()
    Label("Job ID",   job.JobID)
    Label("Job Name", job.JobName)
    Label("Status",   job.Status)
    if job.InProgress {
        Warn("Job läuft noch — laufende Schätzung")
    }
    Label("Duration", fmt.Sprintf("%ds (%.4fh)  →  %s",
        result.DurationSec, result.DurationH, result.FormatDuration()))
    Label("vCPU",     fmt.Sprintf("%.1f", job.VCPU))
    Label("Memory",   fmt.Sprintf("%d MB", job.MemoryMB))
    Separator()
    fmt.Println()
    fmt.Println(styleValue.Render(fmt.Sprintf("--- Estimate (%s) ---", provider)))
    fmt.Println()
    CostLine("CPU Cost",    result.CPUCost)
    CostLine("Memory Cost", result.MemCost)
    Separator()
    CostLine("Total",       result.Total)
    
    // Neue Sektionen
    Separator()
    Label("Cost / Hour", fmt.Sprintf("$%.4f", result.CostPerHour))
    Label("CPU %",       fmt.Sprintf("%.1f%%", result.CPUPct))
    Label("Memory %",    fmt.Sprintf("%.1f%%", result.MemPct))
    Separator()
    
    // Pricing Source Info
    Label("Pricing Source", cfg.Source)
    
    // Static vs Live Vergleich (wenn API genutzt wurde)
    if cfg.Source != "static" {
        staticCfg := pricing.DefaultConfig()
        staticResult := staticCfg.Calculate(job.DurationSec, job.VCPU, job.MemoryMB)
        delta := result.Total - staticResult.Total
        deltaPct := 0.0
        if staticResult.Total > 0 {
            deltaPct = (delta / staticResult.Total) * 100
        }
        deltaSymbol := "📈"
        if delta < 0 {
            deltaSymbol = "📉"
        }
        Label("Static", fmt.Sprintf("$%.4f", staticResult.Total))
        Label("Live", fmt.Sprintf("$%.4f", result.Total))
        Label("Delta", fmt.Sprintf("%s $%+.4f (%+.1f%%)", deltaSymbol, delta, deltaPct))
        Separator()
    }
    
    // Spot Cost
    if result.SpotTotal > 0 {
        spotSaving := (1 - result.SpotTotal/result.Total) * 100
        fmt.Printf("%s %s  (↓%.0f%%)\n",
            styleLabel.Render(fmt.Sprintf("%-20s", "Spot Cost:")),
            styleSpot.Render(fmt.Sprintf("$%.4f", result.SpotTotal)),
            spotSaving)
    }
    
    // Hardware-Empfehlung
    if rec.HasSuggestion {
        Separator()
        fmt.Println()
        fmt.Println(styleRec.Render("Hardware-Empfehlung:"))
        fmt.Printf("  Aktuell:        %.1f vCPU / %d MB\n", rec.CurrentVCPU, rec.CurrentMemMB)
        fmt.Printf("  Empfehlung:     %.1f vCPU / %d MB\n", rec.SuggestedVCPU, rec.SuggestedMemMB)
        if rec.EstimatedSavingPct >= 0 {
            fmt.Printf("  Ersparnis:      ~%.0f%%\n", rec.EstimatedSavingPct)
        } else {
            fmt.Printf("  Mehrkosten:     ~+%.0f%%\n", -rec.EstimatedSavingPct)
        }
        if rec.Reason != "" {
            fmt.Printf("  Basis:          %s\n", rec.Reason)
        }
    }
    
    // Warnung bei CPU/Memory Dominanz
    if result.CPUPct > 70 {
        Warn("CPU dominiert → evtl. overprovisioned")
    } else if result.MemPct > 70 {
        Warn("Memory dominiert → evtl. overprovisioned")
    }
    
    fmt.Println()
}

func PrintCostExplorer(cost *providers.CostInfo, jobName string) {
    fmt.Println()
    Header("Cost Explorer Report")
    fmt.Println()
    Label("Job Tag",  "BatchJob="+jobName)
    Label("Period",   cost.PeriodStart+" → "+cost.PeriodEnd)
    Separator()
    CostLine(fmt.Sprintf("Real Cost (%s)", cost.Unit), cost.Amount)
    fmt.Println()
    Warn(fmt.Sprintf("Cost Explorer hat 24h Verzögerung — Tag BatchJob=%s muss gesetzt sein", jobName))
    fmt.Println()
}

type MultiJobResult struct {
    Queue      string
    Limit      int
    Count      int
    TotalCost  float64
    AvgCost    float64
    MaxCost    float64
    SpotTotal  float64  // 0 wenn nicht verfügbar
    AvgCPUPct  float64
    AvgMemPct  float64
}

type MultiJobJSON struct {
    Queue     string  `json:"queue"`
    Count     int     `json:"count"`
    Total     float64 `json:"total_cost"`
    Average   float64 `json:"average_cost"`
    Max       float64 `json:"max_cost"`
    SpotTotal float64 `json:"spot_total,omitempty"`
    AvgCPUPct float64 `json:"avg_cpu_pct"`
    AvgMemPct float64 `json:"avg_mem_pct"`
}

func PrintLLMAnalysis(text string) {
    fmt.Println()
    fmt.Println(styleHeader.Render("--- KI-Analyse ---"))
    fmt.Println()
    fmt.Println(styleValue.Render(text))
    fmt.Println()
}

func PrintMultiJob(r MultiJobResult, rec string) {
    fmt.Println()
    Header(fmt.Sprintf("Multi Job Analysis (Queue: %s, Limit: %d)", r.Queue, r.Limit))
    fmt.Println()
    Label("Jobs analyzed", fmt.Sprintf("%d", r.Count))
    Separator()
    CostLine("Total Cost",       r.TotalCost)
    CostLine("Average",          r.AvgCost)
    CostLine("Max Job",          r.MaxCost)
    
    if r.SpotTotal > 0 {
        Separator()
        spotSaving := (1 - r.SpotTotal/r.TotalCost) * 100
        fmt.Printf("%s %s  (↓%.0f%%)\n",
            styleLabel.Render(fmt.Sprintf("%-20s", "Spot würde kosten:")),
            styleSpot.Render(fmt.Sprintf("$%.4f", r.SpotTotal)),
            spotSaving)
    }
    
    if rec != "" {
        Separator()
        fmt.Println(styleRec.Render("Hardware-Empfehlung: " + rec))
    }
    
    fmt.Println()
}
