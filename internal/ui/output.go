package ui

import (
    "fmt"

    "github.com/charmbracelet/lipgloss"
    "github.com/lynqtech/batch-cost/internal/pricing"
    "github.com/lynqtech/batch-cost/internal/providers"
)

var (
    styleHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
    styleLabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
    styleCost      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
    styleWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
    styleSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
    styleValue     = lipgloss.NewStyle().Bold(true)
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

func PrintEstimate(job *providers.JobInfo, result pricing.Result, provider string) {
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
