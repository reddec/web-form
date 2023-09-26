package webhook_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/reddec/web-form/internal/schema"
	"github.com/reddec/web-form/internal/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcher_Dispatch(t *testing.T) {
	global, globalCancel := context.WithTimeout(context.Background(), time.Minute)
	defer globalCancel()

	dispatcher := webhook.New(1)
	go func() {
		dispatcher.Run(global)
	}()

	t.Run("defaults", func(t *testing.T) {
		ctx, cancel := createTestContext(global)
		defer cancel()
		server, requests := createTestServer(t)
		defer server.Close()

		err := dispatcher.Dispatch(ctx, schema.Webhook{
			URL: server.URL,
		}, []byte(t.Name()))
		require.NoError(t, err)

		req := requireReceive(t, ctx, requests)
		pd, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte(t.Name()), pd)
		assert.Equal(t, http.MethodPost, req.Method)
	})

	t.Run("custom method", func(t *testing.T) {
		ctx, cancel := createTestContext(global)
		defer cancel()
		server, requests := createTestServer(t)
		defer server.Close()

		err := dispatcher.Dispatch(ctx, schema.Webhook{
			URL:    server.URL,
			Method: http.MethodPut,
		}, []byte(t.Name()))
		require.NoError(t, err)

		req := requireReceive(t, ctx, requests)
		pd, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte(t.Name()), pd)
		assert.Equal(t, http.MethodPut, req.Method)
	})

	t.Run("custom headers", func(t *testing.T) {
		ctx, cancel := createTestContext(global)
		defer cancel()
		server, requests := createTestServer(t)
		defer server.Close()

		err := dispatcher.Dispatch(ctx, schema.Webhook{
			URL: server.URL,
			Headers: map[string]string{
				"Authorization": "foo bar",
			},
		}, []byte(t.Name()))
		require.NoError(t, err)

		req := requireReceive(t, ctx, requests)
		pd, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte(t.Name()), pd)
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "foo bar", req.Header.Get("Authorization"))
	})
}

func createTestServer(t *testing.T) (*httptest.Server, <-chan *http.Request) {
	var arrived = make(chan *http.Request, 1)

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		buffer, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(buffer))
		arrived <- request
		writer.WriteHeader(http.StatusOK)
	}))

	return testServer, arrived
}

func createTestContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 5*time.Second)
}

func requireReceive(t *testing.T, ctx context.Context, requests <-chan *http.Request) *http.Request {
	select {
	case v, ok := <-requests:
		require.True(t, ok)
		return v
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
		panic("finished")
	}
}
