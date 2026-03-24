#!/usr/bin/env bash

# -------------------------------
# Config
# -------------------------------

USE_PRICING_API="${USE_PRICING_API:-1}"
AWS_REGION="${AWS_REGION:-eu-central-1}"
CACHE_FILE="${HOME}/.cache/batch-cost-pricing.env"
CACHE_TTL_MIN=60
DEBUG_PRICING="${DEBUG_PRICING:-0}"

# Default fallback (Fargate x86)
DEFAULT_VCPU_PRICE=0.04048
DEFAULT_GB_PRICE=0.004445

# Export für Main Script
PRICING_SOURCE="unknown"

# -------------------------------
# Logging helper
# -------------------------------

warn_msg() {
  if command -v warn &>/dev/null; then
    warn "$@"
  else
    echo "WARN: $*" >&2
  fi
}

# -------------------------------
# Cache helpers
# -------------------------------

ensure_cache_dir() {
  mkdir -p "$(dirname "$CACHE_FILE")"
}

cache_valid() {
  [[ -f "$CACHE_FILE" ]] || return 1

  local now file_time age
  now=$(date +%s)
  file_time=$(stat -f %m "$CACHE_FILE" 2>/dev/null || stat -c %Y "$CACHE_FILE")

  age=$(( (now - file_time) / 60 ))

  [[ "$DEBUG_PRICING" == "1" ]] && echo "Cache age: ${age} min (ttl=$CACHE_TTL_MIN)" >&2

  (( age >= 0 && age < CACHE_TTL_MIN ))
}

load_cache() {
  if ! source "$CACHE_FILE" 2>/dev/null; then
    warn_msg "Cache beschädigt → wird neu erstellt"
    return 1
  fi

  if [[ -z "$PRICE_PER_VCPU_HOUR" || -z "$PRICE_PER_GB_HOUR" ]]; then
    warn_msg "Cache unvollständig → wird neu erstellt"
    return 1
  fi

  PRICING_SOURCE="cache"
  return 0
}

save_cache() {
  ensure_cache_dir
  cat <<EOF > "$CACHE_FILE"
PRICE_PER_VCPU_HOUR=$PRICE_PER_VCPU_HOUR
PRICE_PER_GB_HOUR=$PRICE_PER_GB_HOUR
EOF
}

# -------------------------------
# AWS Pricing API
# -------------------------------

get_price_from_api() {
  local pattern="$1"

  aws pricing get-products \
    --region us-east-1 \
    --service-code AmazonECS \
    --no-cli-pager \
    --cli-connect-timeout 2 \
    --cli-read-timeout 5 \
    --filters \
      Type=TERM_MATCH,Field=regionCode,Value="$AWS_REGION" \
    --output json 2>/dev/null | \
  jq -r "
    .PriceList[]
    | fromjson
    | select(.product.attributes.usagetype | contains(\"$pattern\"))
    | .terms.OnDemand[]
    | .priceDimensions[]
    | .pricePerUnit.USD
  " | head -n1
}

# -------------------------------
# Pricing Setup
# -------------------------------

set_static_pricing() {
  PRICE_PER_VCPU_HOUR="$DEFAULT_VCPU_PRICE"
  PRICE_PER_GB_HOUR="$DEFAULT_GB_PRICE"
  PRICING_SOURCE="static"
}

load_pricing() {
  echo "[DEBUG] enter load_pricing"

  if [[ "$USE_PRICING_API" != "1" ]]; then
    echo "[DEBUG] USE_PRICING_API disabled → static"
    set_static_pricing
    return
  fi

  echo "[DEBUG] checking cache..."

  if cache_valid; then
    echo "[DEBUG] cache_valid = true"

    if load_cache; then
      echo "[DEBUG] load_cache = true"
      echo "[DEBUG] using cache → exit load_pricing"
      return
    else
      echo "[DEBUG] load_cache = false"
    fi
  else
    echo "[DEBUG] cache_valid = false"
  fi

  echo "[DEBUG] fetching pricing from API..."

  local vcpu_price mem_price

  vcpu_price=$(get_price_from_api "Fargate-vCPU" || true)
  mem_price=$(get_price_from_api "Fargate-GB" || true)

  vcpu_price="${vcpu_price:-}"
  mem_price="${mem_price:-}"

  echo "[DEBUG] API results:"
  echo "[DEBUG] vcpu_price=$vcpu_price"
  echo "[DEBUG] mem_price=$mem_price"

  [[ "$vcpu_price" == "null" || -z "$vcpu_price" ]] && vcpu_price=""
  [[ "$mem_price" == "null" || -z "$mem_price" ]] && mem_price=""

  if [[ -n "$vcpu_price" && -n "$mem_price" ]]; then
    echo "[DEBUG] API success → saving cache"
    PRICE_PER_VCPU_HOUR="$vcpu_price"
    PRICE_PER_GB_HOUR="$mem_price"
    PRICING_SOURCE="api"
    save_cache
  else
    echo "[DEBUG] API failed → fallback"
    warn_msg "Pricing API fehlgeschlagen → fallback auf defaults"
    set_static_pricing
  fi

  echo "[DEBUG] exit load_pricing"
}

# -------------------------------
# Cost Calculation
# -------------------------------

calc_cost_raw() {
  local duration_sec="$1"
  local vcpu="$2"
  local memory_mb="$3"

  local duration_h memory_gb

  duration_h=$(echo "scale=6; $duration_sec / 3600" | bc)
  memory_gb=$(echo "scale=6; $memory_mb / 1024" | bc)

  local cpu_cost mem_cost total cost_per_hour

  cpu_cost=$(echo "$vcpu * $duration_h * $PRICE_PER_VCPU_HOUR" | bc)
  mem_cost=$(echo "$memory_gb * $duration_h * $PRICE_PER_GB_HOUR" | bc)
  total=$(echo "$cpu_cost + $mem_cost" | bc)

  cost_per_hour=$(echo "$vcpu * $PRICE_PER_VCPU_HOUR + $memory_gb * $PRICE_PER_GB_HOUR" | bc)

  echo "$cpu_cost|$mem_cost|$total|$cost_per_hour"
}
