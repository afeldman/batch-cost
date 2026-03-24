#!/usr/bin/env bash
# Output helpers — source this file

# Prüft ob gum verfügbar ist, Fallback auf plain echo
GUM_AVAILABLE=0
command -v gum &>/dev/null && GUM_AVAILABLE=1

format_duration() {
  local sec="$1"
  local h=$(( sec / 3600 ))
  local m=$(( (sec % 3600) / 60 ))
  local s=$(( sec % 60 ))
  printf "%dh %02dm %02ds" "$h" "$m" "$s"
}

header() {
  local text="$1"
  if [[ "$GUM_AVAILABLE" -eq 1 ]]; then
    gum style --bold --foreground 212 "=== $text ==="
  else
    echo "=== $text ==="
  fi
}

color_delta() {
  local value="$1"
  local pct="$2"

  [[ "$value" =~ ^\..* ]] && value="0$value"

  # Vergleich einmal berechnen
  local cmp
  cmp=$(awk -v v="$value" 'BEGIN {
    if (v > 0) print 1;
    else if (v < 0) print -1;
    else print 0;
  }')

  # Sign + Emoji
  local sign=""
  [[ "$cmp" == "1" ]] && sign="+"

  local icon=""
  [[ "$cmp" == "1" ]] && icon="📈"
  [[ "$cmp" == "-1" ]] && icon="📉"

  local text="$icon \$$value (${sign}${pct}%)"

  if [[ "$GUM_AVAILABLE" -eq 1 ]]; then
    if [[ "$cmp" == "1" ]]; then
      gum style --foreground 196 --bold "$text"
    elif [[ "$cmp" == "-1" ]]; then
      gum style --foreground 82 --bold "$text"
    else
      echo "$text"
    fi
  else
    echo "$text"
  fi
}

label() {
  # label "Key" "Value"
  local key="$1" val="$2"
  if [[ "$GUM_AVAILABLE" -eq 1 ]]; then
    printf "%s %s\n" \
      "$(gum style --foreground 240 "$key:")" \
      "$(gum style --bold "$val")"
  else
    printf "%-20s %s\n" "$key:" "$val"
  fi
}

cost_line() {
  # cost_line "CPU Cost" "0.42"
  local key="$1" val="$2"
  if [[ "$GUM_AVAILABLE" -eq 1 ]]; then
    printf "%s %s\n" \
      "$(gum style --foreground 240 "$key:")" \
      "$(gum style --bold --foreground 82 "\$$val")"
  else
    printf "%-20s \$%s\n" "$key:" "$val"
  fi
}

warn() {
  if [[ "$GUM_AVAILABLE" -eq 1 ]]; then
    gum style --foreground 214 "⚠  $*"
  else
    echo "WARN: $*" >&2
  fi
}

err() {
  if [[ "$GUM_AVAILABLE" -eq 1 ]]; then
    gum style --foreground 196 "✗  $*"
  else
    echo "ERROR: $*" >&2
  fi
  exit 1
}

separator() {
  if [[ "$GUM_AVAILABLE" -eq 1 ]]; then
    gum style --foreground 240 "$(printf '─%.0s' {1..40})"
  else
    echo "----------------------------------------"
  fi
}
