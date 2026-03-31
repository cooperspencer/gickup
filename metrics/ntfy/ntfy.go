package ntfy

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cooperspencer/gickup/types"
)

func Notify(msg string, config types.PushConfig) error {
	url := config.Url

	payload := strings.NewReader(msg)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, url, payload)

	req.Header.Add("Content-Type", "text/plain")
	req.Header.Add("Title", "Backup done")

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

	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("received status %d from %s", res.StatusCode, config.Url)
	}

	return nil
}
