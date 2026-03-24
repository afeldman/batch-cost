package pricing

import "fmt"

type Recommendation struct {
    CurrentVCPU        float64
    CurrentMemMB       int64
    SuggestedVCPU      float64
    SuggestedMemMB     int64
    EstimatedSavingPct float64
    Reason             string
    HasSuggestion      bool
}

// Recommend gibt eine Hardware-Empfehlung für einen einzelnen Job.
func Recommend(vcpu float64, memMB int64, cpuPct float64, memPct float64) Recommendation {
    r := Recommendation{CurrentVCPU: vcpu, CurrentMemMB: memMB}

    sugVCPU := vcpu
    sugMem := memMB
    var reasons []string

    // CPU: nur reduzieren wenn wirklich unterausgelastet
    if cpuPct < 40 {
        sugVCPU = maxFloat(0.25, vcpu/2)
        reasons = append(reasons, fmt.Sprintf("CPU %.0f%% → vCPU halbieren", cpuPct))
    } else if cpuPct > 70 && memPct < 50 {
        sugVCPU = vcpu * 2
        reasons = append(reasons, fmt.Sprintf("CPU %.0f%% → vCPU erhöhen", cpuPct))
    }

    // Memory: nur reduzieren wenn tatsächlich weniger als Minimum möglich
    if memPct < 40 {
        newMem := maxInt64(512, memMB/2)
        if newMem < memMB { // nur als Empfehlung wenn wirklich kleiner
            sugMem = newMem
            reasons = append(reasons, fmt.Sprintf("Memory %.0f%% → RAM halbieren", memPct))
        }
    } else if memPct > 70 && cpuPct < 50 {
        sugMem = memMB * 2
        reasons = append(reasons, fmt.Sprintf("Memory %.0f%% → RAM erhöhen", memPct))
    }

    r.SuggestedVCPU = sugVCPU
    r.SuggestedMemMB = sugMem

    if sugVCPU != vcpu || sugMem != memMB {
        r.HasSuggestion = true
        oldRes := vcpu + float64(memMB)/1024
        newRes := sugVCPU + float64(sugMem)/1024
        if oldRes > 0 {
            r.EstimatedSavingPct = (1 - newRes/oldRes) * 100
        }
        if len(reasons) > 0 {
            r.Reason = reasons[0]
            for _, rr := range reasons[1:] {
                r.Reason += "; " + rr
            }
        }
    }
    return r
}

// RecommendMulti gibt eine Text-Empfehlung basierend auf Durchschnittswerten mehrerer Jobs.
// Gibt leeren String zurück wenn keine Empfehlung.
func RecommendMulti(avgCPUPct, avgMemPct float64) string {
    var msgs []string
    if avgCPUPct < 40 {
        msgs = append(msgs, fmt.Sprintf("Ø CPU %.0f%% → vCPU reduzieren", avgCPUPct))
    }
    if avgMemPct < 40 {
        msgs = append(msgs, fmt.Sprintf("Ø Memory %.0f%% → RAM reduzieren", avgMemPct))
    }
    if avgCPUPct > 70 {
        msgs = append(msgs, fmt.Sprintf("Ø CPU %.0f%% → vCPU erhöhen", avgCPUPct))
    }
    if avgMemPct > 70 {
        msgs = append(msgs, fmt.Sprintf("Ø Memory %.0f%% → RAM erhöhen", avgMemPct))
    }
    if len(msgs) == 0 {
        return ""
    }
    result := msgs[0]
    for _, m := range msgs[1:] {
        result += "; " + m
    }
    return result
}

func maxFloat(a, b float64) float64 {
    if a > b {
        return a
    }
    return b
}

func maxInt64(a, b int64) int64 {
    if a > b {
        return a
    }
    return b
}
