package pricing

import "fmt"

type Result struct {
	DurationSec int64
	DurationH   float64
	CPUCost     float64
	MemCost     float64
	Total       float64
	SpotTotal   float64 // 0 wenn kein Spot-Preis
	CPUPct      float64 // Anteil CPU an Gesamtkosten (0-100)
	MemPct      float64 // Anteil Memory an Gesamtkosten (0-100)
	CostPerHour float64
}

func (r Result) FormatDuration() string {
	h := r.DurationSec / 3600
	m := (r.DurationSec % 3600) / 60
	s := r.DurationSec % 60
	return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
}

func (c Config) Calculate(durationSec int64, vcpu float64, memoryMB int64) Result {
	// Minimum billing guard
	if durationSec < 0 {
		durationSec = 0
	}
	if durationSec < 60 {
		durationSec = 60
	}

	durationH := float64(durationSec) / 3600.0
	memGB := float64(memoryMB) / 1024.0

	cpuCost := vcpu * durationH * c.PriceVCPUHour
	memCost := memGB * durationH * c.PriceGBHour
	total := cpuCost + memCost

	var spotTotal float64
	if c.SpotPriceVCPUHour > 0 {
		spotTotal = vcpu*durationH*c.SpotPriceVCPUHour + memGB*durationH*c.SpotPriceGBHour
	}

	var cpuPct, memPct float64
	if total > 0 {
		cpuPct = cpuCost / total * 100
		memPct = memCost / total * 100
	}

	costPerHour := vcpu*c.PriceVCPUHour + memGB*c.PriceGBHour

	return Result{
		DurationSec: durationSec,
		DurationH:   durationH,
		CPUCost:     cpuCost,
		MemCost:     memCost,
		Total:       total,
		SpotTotal:   spotTotal,
		CPUPct:      cpuPct,
		MemPct:      memPct,
		CostPerHour: costPerHour,
	}
}
