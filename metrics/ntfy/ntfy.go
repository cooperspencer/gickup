package ntfy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cooperspencer/gickup/types"
)

func Notify(msg string, config types.PushConfig) error {
	url := config.Url

	payload := strings.NewReader(msg)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, payload)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "text/plain")
	req.Header.Add("Title", "Backup done")
	if config.Email != "" {
		req.Header.Add("Email", config.Email)
	}

	switch {
	case config.Token != "":
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.Token))
	case config.Password != "" && config.User != "":
		req.SetBasicAuth(config.User, config.Password)
	default:
		return fmt.Errorf("neither user, password and token are set")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		statusErr := fmt.Errorf("received status %d from %s", res.StatusCode, config.Url)
		if err := res.Body.Close(); err != nil {
			return errors.Join(statusErr, err)
		}

		return statusErr
	}

	if err := res.Body.Close(); err != nil {
		return err
	}

	return nil
}
