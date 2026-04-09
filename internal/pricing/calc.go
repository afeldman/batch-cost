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
	SpotPriceVCPUHour float64 // 0 = kein Spot-Preis verfügbar
	SpotPriceGBHour   float64
	Source            string // "api" | "cache" | "static"
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
