#!/usr/bin/env bash

validate_number() {
  local name="$1" value="$2"

  if [[ -z "$value" || "$value" == "null" ]]; then
    err "$name ist leer oder null"
  fi

  if ! echo "$value" | grep -Eq '^[0-9]*\.?[0-9]+$'; then
    err "$name ist keine gültige Zahl: $value"
  fi
}

sanitize_duration() {
  local duration="$1"

  # führende 0 fixen (.123 → 0.123)
  [[ "$duration" =~ ^\..* ]] && duration="0$duration"

  # negative verhindern
  if (( $(echo "$duration < 0" | bc -l) )); then
    duration=0
  fi

  # Minimum Billing
  if (( $(echo "$duration < 60" | bc -l) )); then
    duration=60
  fi

  # sauber runden (locale-safe)
  duration=$(LC_NUMERIC=C printf "%.0f" "$duration")

  echo "$duration"
}
