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

Commit `6cc5f5d1` extends that bounded path to transport errors such as
`connect: connection refused`. The first recovery patch only classified hard
failures after an HTTP response body existed; a killed borrowed container can
fail before a response body is available. The follow-up patch classifies the
Go error string too, so a killed borrowed runner uses the short hard-failure
grace instead of the generic borrowed-runner grace.

This is recovery code, not a split-capacity accounting change. It applies to
managed local AI runner containers and can also matter for split-worker hosts
that use the same private Livepeer manager code.

## Sven Kill Test

Controlled test on Sven `dd-us5:8938`:

- old binary `712f03cc`: killing runner `8903` recovered, but only after the
  generic borrowed-runner grace, about `5 minutes` before restart.
- new binary `6cc5f5d1`: killing runner `8903` at `06:54:22 MDT` caused
  connection-refused health failures, container removal/restart at `06:54:31`,
  `LOADING` at `06:54:39`, and `IDLE` at `06:55:09`.

The final post-patch MCP one-hour sample for `dd-us5:8938` showed `51`
streams, `0` errors, `0` swaps, and `0` orchestrator timeouts. This is the
expected recovery behavior for a killed borrowed managed runner.

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
