package llm

import "fmt"

// SingleJobPrompt generiert einen Analyse-Prompt für einen einzelnen Job.
func SingleJobPrompt(jobName, status string, durationSec int64, vcpu float64, memMB int64,
	total, spotTotal, cpuPct, memPct float64, recommendation string) string {
	return fmt.Sprintf(`Du bist ein AWS-Kostenoptimierer. Analysiere diesen AWS Batch Job und gib eine kurze, präzise Bewertung auf Deutsch (max. 3 Sätze):

Job: %s | Status: %s | Dauer: %ds | vCPU: %.1f | Memory: %dMB
Kosten: $%.4f (Spot wäre: $%.4f) | CPU-Anteil: %.0f%% | Memory-Anteil: %.0f%%
Empfehlung: %s

Bewerte: Ist der Job kosteneffizient? Was sollte optimiert werden?`,
		jobName, status, durationSec, vcpu, memMB,
		total, spotTotal, cpuPct, memPct, recommendation)
}

// MultiJobPrompt generiert einen Analyse-Prompt für mehrere Jobs.
func MultiJobPrompt(queue string, count int, total, avg, max, spotTotal, avgCPUPct, avgMemPct float64, recommendation string) string {
	return fmt.Sprintf(`Du bist ein AWS-Kostenoptimierer. Analysiere diese Batch-Job-Statistik und gib eine kurze Bewertung auf Deutsch (max. 3 Sätze):

Queue: %s | %d Jobs analysiert
Gesamtkosten: $%.4f | Ø: $%.4f | Max: $%.4f | Spot wäre: $%.4f
Ø CPU-Auslastung: %.0f%% | Ø Memory-Auslastung: %.0f%%
Empfehlung: %s

Bewerte: Wo steckt das größte Einsparpotenzial?`,
		queue, count, total, avg, max, spotTotal, avgCPUPct, avgMemPct, recommendation)
}
