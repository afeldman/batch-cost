#!/usr/bin/env bash
# Output helpers — source this file

# Prüft ob gum verfügbar ist, Fallback auf plain echo
GUM_AVAILABLE=0
command -v gum &>/dev/null && GUM_AVAILABLE=1

header() {
  local text="$1"
  if [[ "$GUM_AVAILABLE" -eq 1 ]]; then
    gum style --bold --foreground 212 "=== $text ==="
  else
    echo "=== $text ==="
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
