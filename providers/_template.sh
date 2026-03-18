#!/usr/bin/env bash
# Provider Template — kopiere diese Datei für neue Provider
# Benenne sie z.B. gcp.sh oder azure.sh
# Implementiere die folgenden Funktionen:

# Gibt Job-Metadaten als JSON zurück (gleiche Struktur wie aws.sh)
# Felder: job_id, job_name, status, in_progress,
#         duration_sec, vcpu, memory_mb
PROVIDER_describe_job() {
  local job_id="$1"
  # TODO: implementieren
  err "Provider nicht implementiert"
}

# Gibt echte Kosten als JSON zurück
# Felder: amount, unit, period_start, period_end
PROVIDER_cost_explorer() {
  local job_name="$1"
  # TODO: implementieren
  err "Provider nicht implementiert"
}
