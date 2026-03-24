package pricing

import (
    "fmt"
    "os"
    "strconv"

    "github.com/BurntSushi/toml"
    "github.com/afeldman/batch-cost/internal/llm"
)

const (
    DefaultPriceVCPUHour float64 = 0.04048
    DefaultPriceGBHour   float64 = 0.004445
)

type Config struct {
    PriceVCPUHour     float64
    PriceGBHour       float64
    SpotPriceVCPUHour float64  // 0 = kein Spot-Preis verfügbar
    SpotPriceGBHour   float64
    Source            string   // "api" | "cache" | "static"
}

type pricingSection struct {
    PriceVCPUHour   float64 `toml:"price_vcpu_hour"`
    PriceGBHour     float64 `toml:"price_gb_hour"`
    UsePricingAPI   bool    `toml:"use_pricing_api"`
    CacheTTLMinutes int     `toml:"cache_ttl_minutes"`
}

type cacheSection struct {
    Dir string `toml:"dir"`
}

type tomlFile struct {
    Pricing pricingSection `toml:"pricing"`
    Cache   cacheSection   `toml:"cache"`
    LLM     llm.Config     `toml:"llm"`
}

// DefaultConfig gibt die Standard-Pricing-Konfiguration zurück.
func DefaultConfig() Config {
    return Config{
        PriceVCPUHour: DefaultPriceVCPUHour,
        PriceGBHour:   DefaultPriceGBHour,
        Source:        "static",
    }
}

// LoadOptions liest PricingOptions + Basis-Config aus TOML/Env.
func LoadOptions(path string) (Config, PricingOptions, llm.Config, error) {
    cfg := DefaultConfig()
    opts := DefaultOptions()
    llmCfg := llm.Config{
        Enabled:  false,
        Endpoint: "http://localhost:1234/v1",
        Model:    "",
        APIKey:   "lm-studio",
        TimeoutS: 30,
        Local: llm.LocalConfig{
            Enabled:   false,
            ModelRepo: "noeum/noeum-1-nano-base",
            Port:      2510,
            ConfigDir: "",
        },
    }

    // Env-Vars für Preise
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
            return cfg, opts, llmCfg, fmt.Errorf("pricing.toml (%s): %w", path, err)
        }
        if f.Pricing.PriceVCPUHour > 0 {
            cfg.PriceVCPUHour = f.Pricing.PriceVCPUHour
        }
        if f.Pricing.PriceGBHour > 0 {
            cfg.PriceGBHour = f.Pricing.PriceGBHour
        }
        opts.UsePricingAPI = f.Pricing.UsePricingAPI
        if f.Pricing.CacheTTLMinutes > 0 {
            opts.CacheTTLMinutes = f.Pricing.CacheTTLMinutes
        }
        opts.CacheDir = f.Cache.Dir
        
        // LLM-Konfiguration
        llmCfg = f.LLM
    }

    return cfg, opts, llmCfg, nil
}

type Result struct {
    DurationSec int64
    DurationH   float64
    CPUCost     float64
    MemCost     float64
    Total       float64
    SpotTotal   float64  // 0 wenn kein Spot-Preis
    CPUPct      float64  // Anteil CPU an Gesamtkosten (0-100)
    MemPct      float64  // Anteil Memory an Gesamtkosten (0-100)
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
