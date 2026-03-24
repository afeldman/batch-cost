package pricing

import (
    "context"
    "fmt"
)

// ResolvePrices ermittelt aktuelle Preise basierend auf PricingOptions.
// Aufruf-Sequenz in cmd/root.go:
//   cfg, opts, err := pricing.LoadOptions(flagConfig)
//   cfg, err = pricing.ResolvePrices(ctx, cfg, opts, region)
func ResolvePrices(ctx context.Context, base Config, opts PricingOptions, region string) (Config, error) {
    if !opts.UsePricingAPI {
        base.Source = "static"
        return base, nil
    }

    cachePath := CachePath(opts.CacheDir)

    if CacheValid(cachePath, opts.CacheTTLMinutes) {
        if cached, ok := LoadCache(cachePath); ok {
            return cached, nil
        }
    }

    // API abrufen (Aufrufer zeigt Spinner)
    live, err := FetchFargatePrices(ctx, region)
    if err != nil {
        fmt.Printf("⚠  Pricing API fehlgeschlagen (%v) → statische Preise\n", err)
        base.Source = "static"
        return base, nil
    }

    _ = SaveCache(cachePath, live)
    return live, nil
}
