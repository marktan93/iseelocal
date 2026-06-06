package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"iseelocal/internal/shared/contracts"
	"iseelocal/internal/shared/validation"
)

func CheckHTTPTarget(ctx context.Context, target contracts.LocalTarget, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if err := validation.ValidateLocalTarget(target, false); err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := "http://" + net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: timeout}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("local target %s is not reachable: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("local target %s returned %s", url, res.Status)
	}
	return nil
}
