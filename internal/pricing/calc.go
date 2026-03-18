package pricing

import (
    "fmt"
    "os"
    "strconv"

    "github.com/BurntSushi/toml"
)

const (
    DefaultPriceVCPUHour float64 = 0.04048
    DefaultPriceGBHour   float64 = 0.004445
)

type Config struct {
    PriceVCPUHour float64 `toml:"price_vcpu_hour"`
    PriceGBHour   float64 `toml:"price_gb_hour"`
}

type tomlFile struct {
    Pricing Config `toml:"pricing"`
}

// DefaultConfig gibt die Standard-Pricing-Konfiguration zurück.
func DefaultConfig() Config {
    return Config{
        PriceVCPUHour: DefaultPriceVCPUHour,
        PriceGBHour:   DefaultPriceGBHour,
    }
}

// LoadConfig lädt Pricing-Konfiguration.
// Priorität: TOML-Datei > Env-Vars (PRICE_PER_VCPU_HOUR, PRICE_PER_GB_HOUR) > Defaults
// path: leer = Auto-Discovery (./pricing.toml, ~/.config/batch-cost/pricing.toml)
func LoadConfig(path string) (Config, error) {
    cfg := DefaultConfig()

    // Env-Vars
    if v := os.Getenv("PRICE_PER_VCPU_HOUR"); v != "" {
        if f, err := strconv.ParseFloat(v, 64); err == nil {
            cfg.PriceVCPUHour = f
        }
    }
    if v := os.Getenv("PRICE_PER_GB_HOUR"); v != "" {
        if f, err := strconv.ParseFloat(v, 64); err == nil {
            cfg.PriceGBHour = f
        }
    }

    // Auto-Discovery wenn kein expliziter Pfad
    if path == "" {
        for _, candidate := range []string{
            "pricing.toml",
            os.ExpandEnv("$HOME/.config/batch-cost/pricing.toml"),
        } {
            if _, err := os.Stat(candidate); err == nil {
                path = candidate
                break
            }
        }
    }

    // TOML laden — überschreibt Env-Vars und Defaults
    if path != "" {
        var f tomlFile
        if _, err := toml.DecodeFile(path, &f); err != nil {
            return cfg, fmt.Errorf("pricing.toml (%s): %w", path, err)
        }
        if f.Pricing.PriceVCPUHour > 0 {
            cfg.PriceVCPUHour = f.Pricing.PriceVCPUHour
        }
        if f.Pricing.PriceGBHour > 0 {
            cfg.PriceGBHour = f.Pricing.PriceGBHour
        }
    }

    return cfg, nil
}

type Result struct {
    DurationSec int64
    DurationH   float64
    CPUCost     float64
    MemCost     float64
    Total       float64
}

func (r Result) FormatDuration() string {
    h := r.DurationSec / 3600
    m := (r.DurationSec % 3600) / 60
    s := r.DurationSec % 60
    return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
}

func (c Config) Calculate(durationSec int64, vcpu float64, memoryMB int64) Result {
    durationH := float64(durationSec) / 3600.0
    memGB := float64(memoryMB) / 1024.0

    cpuCost := vcpu * durationH * c.PriceVCPUHour
    memCost := memGB * durationH * c.PriceGBHour

    return Result{
        DurationSec: durationSec,
        DurationH:   durationH,
        CPUCost:     cpuCost,
        MemCost:     memCost,
        Total:       cpuCost + memCost,
    }
}
