package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
)

const MaxAttempts = 3

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	FinalURL   string
}

// Do retries transient GET failures and statuses. The caller must close the response body.
func Do(client *http.Client, req *http.Request, log *logger.Logger) (*http.Response, error) {
	return run(client, req, log, false, func(resp *http.Response) (*http.Response, bool, error) {
		return resp, false, nil
	})
}

// ReadAll also retries interrupted response-body reads.
func ReadAll(client *http.Client, req *http.Request, log *logger.Logger) (Response, error) {
	return run(client, req, log, true, func(resp *http.Response) (Response, bool, error) {
		result := Response{StatusCode: resp.StatusCode, Header: resp.Header}
		if resp.Request != nil && resp.Request.URL != nil {
			result.FinalURL = resp.Request.URL.String()
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return Response{}, resp.StatusCode == http.StatusOK,
				fmt.Errorf("failed to read HTTP response: %w", err)
		}
		result.Body = body
		return result, false, nil
	})
}

// DownloadToFile streams a GET into a temporary file, retries interrupted reads,
// and only replaces dest after a complete download.
func DownloadToFile(client *http.Client, req *http.Request, dest string, log *logger.Logger) error {
	_, err := run(client, req, log, true, func(resp *http.Response) (struct{}, bool, error) {
		if resp.StatusCode != http.StatusOK {
			return struct{}{}, false, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, req.URL)
		}

		out, err := os.CreateTemp(filepath.Dir(dest), ".goweeb-*.part")
		if err != nil {
			return struct{}{}, false, fmt.Errorf("failed to create temporary image: %w", err)
		}
		tmpPath := out.Name()
		defer os.Remove(tmpPath)

		_, copyErr := io.Copy(out, resp.Body)
		closeErr := out.Close()
		if copyErr != nil {
			var pathErr *os.PathError
			retry := !errors.As(copyErr, &pathErr)
			return struct{}{}, retry, fmt.Errorf("failed to download %s: %w", req.URL, copyErr)
		}
		if closeErr != nil {
			return struct{}{}, false, fmt.Errorf("failed to close temporary image: %w", closeErr)
		}
		if err := os.Rename(tmpPath, dest); err != nil {
			return struct{}{}, false, fmt.Errorf("failed to save image: %w", err)
		}
		return struct{}{}, false, nil
	})
	return err
}

func run[T any](
	client *http.Client,
	req *http.Request,
	log *logger.Logger,
	closeBody bool,
	handle func(*http.Response) (T, bool, error),
) (T, error) {
	var zero T
	if client == nil || req == nil || req.URL == nil {
		return zero, errors.New("nil HTTP client or request")
	}
	if req.Method != http.MethodGet || req.Body != nil {
		return zero, errors.New("HTTP retry only supports GET requests without a body")
	}

	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		if err := req.Context().Err(); err != nil {
			return zero, err
		}
		resp, err := client.Do(req.Clone(req.Context()))
		if err != nil {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			if attempt == MaxAttempts || req.Context().Err() != nil || errors.Is(err, context.Canceled) {
				return zero, fmt.Errorf("GET %s failed after %d attempt(s): %w", req.URL, attempt, err)
			}
			debug(log, "HTTP GET %s failed (%v), retrying (%d/%d)\n", req.URL, err, attempt+1, MaxAttempts)
			if err := wait(req.Context(), retryDelay(attempt)); err != nil {
				return zero, err
			}
			continue
		}

		if retryableStatus(resp.StatusCode) && attempt < MaxAttempts {
			delay, allowed := statusRetryDelay(resp, attempt)
			if allowed {
				resp.Body.Close()
				debug(log, "HTTP GET %s returned %d, retrying (%d/%d)\n", req.URL, resp.StatusCode, attempt+1, MaxAttempts)
				if err := wait(req.Context(), delay); err != nil {
					return zero, err
				}
				continue
			}
		}

		value, retryRead, handleErr := handle(resp)
		if closeBody {
			resp.Body.Close()
		}
		if handleErr != nil {
			if retryRead && attempt < MaxAttempts && req.Context().Err() == nil {
				if !closeBody {
					resp.Body.Close()
				}
				debug(log, "HTTP GET %s body read failed (%v), retrying (%d/%d)\n", req.URL, handleErr, attempt+1, MaxAttempts)
				if err := wait(req.Context(), retryDelay(attempt)); err != nil {
					return zero, err
				}
				continue
			}
			return zero, fmt.Errorf("GET %s failed after %d attempt(s): %w", req.URL, attempt, handleErr)
		}
		if attempt > 1 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			debug(log, "HTTP GET %s succeeded on attempt %d/%d\n", req.URL, attempt, MaxAttempts)
		}
		return value, nil
	}
	return zero, errors.New("HTTP retry attempts exhausted")
}

func retryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests ||
		(status >= 500 && status <= 599 && status != http.StatusNotImplemented && status != http.StatusHTTPVersionNotSupported)
}

func retryDelay(attempt int) time.Duration {
	return time.Duration(1<<(attempt-1)) * 250 * time.Millisecond
}

// Avoid blocking the CLI on a very long Retry-After. In that case leave
// the status unchanged and let the caller report it.
func statusRetryDelay(resp *http.Response, attempt int) (time.Duration, bool) {
	fallback := retryDelay(attempt)
	if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode != http.StatusServiceUnavailable {
		return fallback, true
	}
	raw := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if raw == "" {
		return fallback, true
	}
	var d time.Duration
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil && seconds >= 0 {
		if seconds > 10 {
			return 0, false
		}
		d = time.Duration(seconds) * time.Second
	} else if date, err := http.ParseTime(raw); err == nil {
		d = time.Until(date)
	} else {
		return fallback, true
	}
	if d > 10*time.Second {
		return 0, false
	}
	if d < 0 {
		d = 0
	}
	return d, true
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func debug(log *logger.Logger, format string, args ...any) {
	if log != nil {
		log.Debug(format, args...)
	}
}
