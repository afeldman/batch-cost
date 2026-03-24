calc_cost_with_prices() {
  local duration="$1"
  local vcpu="$2"
  local memory="$3"
  local cpu_price="$4"
  local mem_price="$5"

  local duration_h memory_gb

  duration_h=$(echo "scale=6; $duration / 3600" | bc)
  memory_gb=$(echo "scale=6; $memory / 1024" | bc)

  local cpu_cost mem_cost total

  cpu_cost=$(echo "$vcpu * $duration_h * $cpu_price" | bc)
  mem_cost=$(echo "$memory_gb * $duration_h * $mem_price" | bc)
  total=$(echo "$cpu_cost + $mem_cost" | bc)

  echo "$total"
}

aws_list_jobs() {
  local limit="$1"

  for status in SUCCEEDED FAILED RUNNING; do
    aws batch list-jobs \
      --job-queue "$JOB_QUEUE" \
      --job-status "$status" \
      --output json
  done | jq -r '.jobSummaryList[].jobId' | head -n "$limit"
}


