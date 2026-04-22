package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExternalOnlyWorkerUsesExternalCapacityWithoutDocker(t *testing.T) {
	runner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"IDLE"}`))
		case "/hardware/info":
			_, _ = w.Write([]byte(`{"pipeline":"live-video-to-video","model_id":"streamdiffusion-sdxl","gpu_info":{}}`))
		case "/version":
			_, _ = w.Write([]byte(`{"pipeline":"live-video-to-video","model_id":"streamdiffusion-sdxl","version":"test-runner"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(runner.Close)

	w := NewExternalOnlyWorker()
	ctx := context.Background()
	pipeline := "live-video-to-video"
	modelID := "streamdiffusion-sdxl"

	require.NoError(t, w.EnsureImageAvailable(ctx, pipeline, modelID))
	require.NoError(t, w.Warm(ctx, pipeline, modelID, RunnerEndpoint{URL: runner.URL}, nil))
	require.True(t, w.HasCapacity(pipeline, modelID))

	capacity := w.GetLiveAICapacity(pipeline, modelID)
	require.Equal(t, 0, capacity.ContainersInUse)
	require.Equal(t, 1, capacity.ContainersIdle)

	container, err := w.borrowContainer(ctx, pipeline, modelID)
	require.NoError(t, err)
	require.Equal(t, External, container.Type)
	require.Equal(t, runner.URL, container.Endpoint.URL)

	require.NoError(t, w.Stop(ctx))
	require.False(t, w.HasCapacity(pipeline, modelID))
}

func TestExternalOnlyWorkerRejectsManagedWarmWithoutDocker(t *testing.T) {
	w := NewExternalOnlyWorker()

	err := w.Warm(context.Background(), "text-to-image", "model", RunnerEndpoint{}, nil)
	require.ErrorContains(t, err, "without Docker manager")
}
