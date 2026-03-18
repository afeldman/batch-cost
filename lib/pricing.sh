#!/usr/bin/env bash
# Pricing helpers — source this file

# AWS Fargate on-demand Preise (eu-central-1, Stand 2024)
# Überschreibbar via Umgebungsvariablen
PRICE_PER_VCPU_HOUR="${PRICE_PER_VCPU_HOUR:-0.04048}"
PRICE_PER_GB_HOUR="${PRICE_PER_GB_HOUR:-0.004445}"

# Berechnet Kosten mit bc (high precision)
# calc_cost <duration_seconds> <vcpu> <memory_mb>
# Gibt drei Zeilen aus: cpu_cost mem_cost total
calc_cost() {
  local duration_sec="$1"
  local vcpu="$2"
  local memory_mb="$3"

  local duration_h
  duration_h=$(echo "scale=6; $duration_sec / 3600" | bc)

  local memory_gb
  memory_gb=$(echo "scale=6; $memory_mb / 1024" | bc)

  local cpu_cost
  cpu_cost=$(echo "scale=6; $vcpu * $duration_h * $PRICE_PER_VCPU_HOUR" | bc)

  local mem_cost
  mem_cost=$(echo "scale=6; $memory_gb * $duration_h * $PRICE_PER_GB_HOUR" | bc)

  local total
  total=$(echo "scale=6; $cpu_cost + $mem_cost" | bc)

  # Runden auf 4 Dezimalstellen für Anzeige
  # LC_NUMERIC=C für konsistente Dezimalpunkte
  cpu_cost=$(LC_NUMERIC=C printf "%.4f" "$cpu_cost")
  mem_cost=$(LC_NUMERIC=C printf "%.4f" "$mem_cost")
  total=$(LC_NUMERIC=C printf "%.4f" "$total")

  echo "$cpu_cost"
  echo "$mem_cost"
  echo "$total"
}

# Hilfsfunktion: Sekunden → "Xh Ym Zs"
format_duration() {
  local sec="$1"
  local h=$(( sec / 3600 ))
  local m=$(( (sec % 3600) / 60 ))
  local s=$(( sec % 60 ))
  printf "%dh %02dm %02ds" "$h" "$m" "$s"
}
