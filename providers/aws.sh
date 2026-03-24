#!/usr/bin/env bash

aws_describe_job() {
  local job_id="$1"

  aws batch describe-jobs --jobs "$job_id" | jq -r '
    .jobs[0] |
    {
      job_id: .jobId,
      job_name: .jobName,
      status: .status,
      vcpu: .container.vcpus,
      memory_mb: .container.memory,
      started_at: .startedAt,
      stopped_at: .stoppedAt,
      compute_environment: .container.instanceType // "FARGATE"
    } |
    .duration_sec = (
      if .stopped_at then
        ((.stopped_at - .started_at) / 1000)
      else
        ((now * 1000 - .started_at) / 1000)
      end
    ) |
    .in_progress = (if .stopped_at then 0 else 1 end)
  '
}

aws_describe_multi_job() {
  local job_id="$1"

  aws batch describe-jobs --jobs "$job_id" --output json | jq -r '
    .jobs[0] |
    {
      job_id: .jobId,
      job_name: .jobName,
      status: .status,

      duration_sec: (
        if (.startedAt and .stoppedAt) then
          ((.stoppedAt - .startedAt) / 1000)
        else
          0
        end
      ),

      vcpu: (
        if .container.resourceRequirements then
          (.container.resourceRequirements[]
            | select(.type=="VCPU")
            | .value) // "0"
        else
          (.container.vcpus // 0)
        end
      ),

      memory_mb: (
        if .container.resourceRequirements then
          (.container.resourceRequirements[]
            | select(.type=="MEMORY")
            | .value) // "0"
        else
          (.container.memory // 0)
        end
      )
    }
  '
}
