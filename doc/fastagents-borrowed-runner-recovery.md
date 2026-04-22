# FastAgents Borrowed Runner Recovery

Status: operational note for the FastAgents private Livepeer fork.

## Problem

Commit `f9a7b36c` protected active live-video runners by ignoring `/health`
HTTP failures while a managed runner container was borrowed. That avoided
killing paid streams during short transient health-check timeouts, but it also
removed the normal Docker manager recovery path.

On Sven, two SDXL runners stayed borrowed forever while `/health` returned:

```text
{"detail":{"msg":"Failed to retrieve pipeline status."}}
```

The affected containers remained `docker ps` healthy enough to look `Up`, but
their GPUs were cold and they never returned to the idle pool. The front then
advertised fewer real lanes than intended and could emit `503 insufficient
capacity` while stale containers occupied the lane accounting.

## Fix

Commit `712f03cc` keeps the active-stream protection bounded:

- short borrowed-runner health failures are still tolerated
- hard failure bodies such as `Failed to retrieve pipeline status` are tracked
  separately
- a persistent hard borrowed failure after the grace window is routed through
  the existing managed-container destroy and warm path
- `OK` or `IDLE` health resets the borrowed failure counters

This is recovery code, not a split-capacity accounting change. It applies to
managed local AI runner containers and can also matter for split-worker hosts
that use the same private Livepeer manager code.

## Required Audit

After deploying this code, do not trust `docker ps` alone. Verify all layers:

- running Livepeer process hash matches the installed binary hash
- every runner port reports `OK`, `IDLE`, or expected `LOADING`
- GPU VRAM residency matches the model profile for every advertised lane
- Docker restart counts and OOM flags are clean after warm-up
- filtered orch logs have no repeated borrowed-runner health failures
- real traffic produces request and payment-ticket log lines
- MCP/exporter shows the URL with nonzero capacity and no new error burst

The expected Sven SDXL profile is four warm runners on ports `8900-8903`, each
using about `23 GiB` VRAM.
