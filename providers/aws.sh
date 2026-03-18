#!/usr/bin/env bash
# AWS Provider — source this file
# Requires: aws cli, jq

AWS_REGION="${AWS_REGION:-eu-central-1}"
AWS_PROFILE_FLAG=""
[[ -n "${AWS_PROFILE:-}" ]] && AWS_PROFILE_FLAG="--profile $AWS_PROFILE"

# Wrapper für aws cli Fehlerbehandlung
_aws() {
  aws $AWS_PROFILE_FLAG --region "$AWS_REGION" "$@" 2>/dev/null
}

# Mode 1: Job per ID beschreiben
# Gibt JSON-Objekt zurück mit: job_id, job_name, status,
#   start_time, end_time, duration_sec, vcpu, memory_mb
aws_describe_job() {
  local job_id="$1"

  local raw
  raw=$(_aws batch describe-jobs --jobs "$job_id") \
    || err "AWS Batch describe-jobs fehlgeschlagen"

  local job
  job=$(echo "$raw" | jq '.jobs[0]') \
    || err "Job nicht gefunden: $job_id"

  [[ "$job" == "null" ]] && err "Job nicht gefunden: $job_id"

  local status
  status=$(echo "$job" | jq -r '.status')

  local start_ms end_ms duration_sec
  start_ms=$(echo "$job" | jq -r '.startedAt // empty')
  end_ms=$(echo "$job" | jq -r '.stoppedAt // empty')

  # Job noch aktiv?
  local in_progress=0
  if [[ "$status" == "RUNNING" ]]; then
    in_progress=1
    end_ms=$(date +%s%3N)
    warn "Job läuft noch — Kosten werden bis jetzt geschätzt"
  fi

  [[ -z "$start_ms" ]] && err "Keine Startzeit gefunden (Status: $status)"

  duration_sec=$(( (end_ms - start_ms) / 1000 ))

  local vcpu memory_mb
  vcpu=$(echo "$job" | jq -r '
    .container.resourceRequirements[]?
    | select(.type == "VCPU") | .value // "1"
  ' | head -1)
  memory_mb=$(echo "$job" | jq -r '
    .container.resourceRequirements[]?
    | select(.type == "MEMORY") | .value // "2048"
  ' | head -1)

  # Fallback auf container.vcpus / memory
  [[ -z "$vcpu" || "$vcpu" == "null" ]] && \
    vcpu=$(echo "$job" | jq -r '.container.vcpus // 1')
  [[ -z "$memory_mb" || "$memory_mb" == "null" ]] && \
    memory_mb=$(echo "$job" | jq -r '.container.memory // 2048')

  jq -n \
    --arg job_id     "$(echo "$job" | jq -r '.jobId')" \
    --arg job_name   "$(echo "$job" | jq -r '.jobName')" \
    --arg status     "$status" \
    --argjson in_progress "$in_progress" \
    --argjson duration_sec "$duration_sec" \
    --argjson vcpu        "$vcpu" \
    --argjson memory_mb   "$memory_mb" \
    '{
      job_id: $job_id,
      job_name: $job_name,
      status: $status,
      in_progress: $in_progress,
      duration_sec: $duration_sec,
      vcpu: $vcpu,
      memory_mb: $memory_mb
    }'
}

# Mode 2: Echte Kosten via Cost Explorer
# Gibt Betrag + Währung zurück
aws_cost_explorer() {
  local job_name="$1"

  # Letzten 30 Tage
  local start_date end_date
  end_date=$(date +%Y-%m-%d)
  start_date=$(date -d "-30 days" +%Y-%m-%d 2>/dev/null \
    || date -v -30d +%Y-%m-%d)   # macOS kompatibel

  local result
  result=$(_aws ce get-cost-and-usage \
    --time-period "Start=${start_date},End=${end_date}" \
    --granularity MONTHLY \
    --metrics "UnblendedCost" \
    --filter "{
      \"Tags\": {
        \"Key\": \"BatchJob\",
        \"Values\": [\"${job_name}\"]
      }
    }") || err "Cost Explorer Abfrage fehlgeschlagen (Tag: BatchJob=${job_name})"

  echo "$result" | jq '{
    amount: (.ResultsByTime[0].Total.UnblendedCost.Amount // "0"),
    unit:   (.ResultsByTime[0].Total.UnblendedCost.Unit   // "USD"),
    period_start: "'$start_date'",
    period_end:   "'$end_date'"
  }'
}
