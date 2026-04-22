# FastAgents Split Capacity Contract

Status: operational note for the FastAgents private Livepeer fork.

Split Live AI fronts must expose the same gateway-facing capacity contract as a
local orch:

- `capacity` means currently idle usable containers or worker lanes.
- `capacityInUse` means registered capacity already borrowed by active work.
- `capacity + capacityInUse` must equal the registered warm lane count for the
  model, unless a worker disconnects or is intentionally removed.
- repeated requests for the same live session must reuse the same borrowed
  remote worker and must not decrement capacity again.
- `live-video-to-video` capacity stays borrowed until the stream/session context
  closes; non-live jobs release capacity after the request completes.

The production log line to check on a split front is:

```text
GetLiveAICapacity: pipeline=live-video-to-video modelID=streamdiffusion-sdxl remote=2 idle=2 inUse=0 total=2
```

For production Arbitrum mainnet builds, always build with:

```bash
make livepeer BUILD_TAGS=mainnet,experimental
```

Before installing, run an explicit chain support probe for chain ID `42161`.
A binary built without the `mainnet` tag is dev-only and exits on
`-network arbitrum-one-mainnet`.

The us4 split-front parity fix was deployed from commit
`777c0a1cf1fbcfa9e1408f62b2319ce4dd3dede3`, version
`0.8.10-777c0a1c`, with installed SHA256
`cb0b3ef5af69eb7d8fd88cc4558a76eba5b74fee25247f35abdbf7ebb7078291`.
The exact deployed-code tag is
`fastagents-split-sdxl-us4-front-v0.8.10-777c0a1c`; later branch head
`f514c2e9` is documentation-only, so a live binary reporting
`0.8.10-777c0a1c` is expected and not stale.
