package gotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cooperspencer/gickup/types"
)

func Notify(msg string, config types.PushConfig) error {
	if !strings.HasSuffix(config.Url, "/") {
		config.Url += "/"
	}

	url := fmt.Sprintf("%smessage?token=%s", config.Url, config.Token)

	payload := map[string]string{}
	payload["message"] = msg
	payload["title"] = "Backup done"

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))

	req.Header.Add("Content-Type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("received status %d from %s", res.StatusCode, config.Url)
	}

	return nil
}
