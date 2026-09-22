package cmd

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// slowServer never answers: it waits for the client to give up.
func slowServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv
}

// runCtx is runCmd with a caller-supplied context, for cancellation tests.
func runCtx(t *testing.T, ctx context.Context, baseURL string, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("TBA_AUTH_KEY", "test-key")
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())

	root := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs(append([]string{"--base-url", baseURL}, args...))

	err := Run(ctx, root)
	return outBuf.String(), errBuf.String(), err
}

func TestCancelledContextAbortsARequestPromptly(t *testing.T) {
	srv := slowServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, _, err := runCtx(t, ctx, srv.URL, "status")
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("want an error when the context is cancelled")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error = %v, want one wrapping context.Canceled", err)
		}
		if got := clierr.ExitCode(err); got != clierr.ExitInterrupt {
			t.Errorf("exit code = %d, want %d", got, clierr.ExitInterrupt)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("the command took %v to notice the cancellation", elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the command did not stop when the context was cancelled")
	}
}

func TestAlreadyCancelledContextMakesNoRequest(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := runCtx(t, ctx, srv.URL, "status")
	if err == nil {
		t.Fatal("want an error for a cancelled context")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitInterrupt {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitInterrupt)
	}
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("no request should have been sent, got %v", got)
	}
}

// Commands must pass the context through, not fall back to Background.
func TestCommandsUseTheirContext(t *testing.T) {
	cases := [][]string{
		{"status"},
		{"team", "view", "177"},
		{"event", "matches", "2024cthar"},
		{"event", "insights", "2024cthar"},
		{"district", "list", "--year", "2024"},
		{"insight", "notables", "--year", "2024"},
		{"match", "view", "2024cthar_qm1"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			srv := slowServer(t)
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				_, _, err := runCtx(t, ctx, srv.URL, args...)
				done <- err
			}()

			select {
			case err := <-done:
				if err == nil || !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("error = %v, want one wrapping context.DeadlineExceeded", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("the command ignored its context")
			}
		})
	}
}
