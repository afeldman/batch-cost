package pricing

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

// CachePath gibt den vollständigen Pfad zur Cache-Datei zurück.
func CachePath(dir string) string {
    if dir == "" {
        dir = filepath.Join(os.Getenv("HOME"), ".cache", "batch-cost")
    }
    return filepath.Join(dir, "pricing.env")
}

// CacheValid prüft ob Cache existiert und TTL nicht überschritten.
func CacheValid(path string, ttlMin int) bool {
    info, err := os.Stat(path)
    if err != nil {
        return false
    }
    age := time.Since(info.ModTime()).Minutes()
    return age < float64(ttlMin)
}

// LoadCache liest Preise aus der Cache-Datei.
// Gibt false zurück wenn Datei fehlt, beschädigt oder mandatory Keys fehlen.
func LoadCache(path string) (Config, bool) {
    data, err := os.ReadFile(path)
    if err != nil {
        return Config{}, false
    }
    m := make(map[string]string)
    for _, line := range strings.Split(string(data), "\n") {
        parts := strings.SplitN(line, "=", 2)
        if len(parts) == 2 {
            m[parts[0]] = parts[1]
        }
    }
    // Mandatory keys
    vcpu, ok1 := m["PRICE_PER_VCPU_HOUR"]
    gb, ok2 := m["PRICE_PER_GB_HOUR"]
    if !ok1 || !ok2 {
        return Config{}, false
    }
    cfg := Config{Source: "cache"}
    cfg.PriceVCPUHour, _ = strconv.ParseFloat(vcpu, 64)
    cfg.PriceGBHour, _ = strconv.ParseFloat(gb, 64)
    // Optional spot keys
    if v, ok := m["SPOT_PRICE_VCPU_HOUR"]; ok {
        cfg.SpotPriceVCPUHour, _ = strconv.ParseFloat(v, 64)
    }
    if v, ok := m["SPOT_PRICE_GB_HOUR"]; ok {
        cfg.SpotPriceGBHour, _ = strconv.ParseFloat(v, 64)
    }
    if cfg.PriceVCPUHour == 0 || cfg.PriceGBHour == 0 {
        return Config{}, false
    }
    return cfg, true
}

// SaveCache schreibt Preise in die Cache-Datei.
func SaveCache(path string, cfg Config) error {
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return err
    }
    content := fmt.Sprintf(
        "PRICE_PER_VCPU_HOUR=%f\nPRICE_PER_GB_HOUR=%f\nSPOT_PRICE_VCPU_HOUR=%f\nSPOT_PRICE_GB_HOUR=%f\n",
        cfg.PriceVCPUHour, cfg.PriceGBHour, cfg.SpotPriceVCPUHour, cfg.SpotPriceGBHour,
    )
    return os.WriteFile(path, []byte(content), 0644)
}
