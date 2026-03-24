package pricing

// PricingOptions enthält die Input-Konfiguration (aus TOML/Env).
// Getrennt von Config (reine Preis-Daten).
type PricingOptions struct {
    UsePricingAPI   bool
    CacheTTLMinutes int    // Default: 60
    CacheDir        string // leer = $HOME/.cache/batch-cost/
}

// DefaultOptions gibt sinnvolle Defaults zurück.
func DefaultOptions() PricingOptions {
    return PricingOptions{
        UsePricingAPI:   true,
        CacheTTLMinutes: 60,
        CacheDir:        "",
    }
}
