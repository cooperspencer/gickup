package apprise

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/cooperspencer/gickup/types"
)

type Request struct {
	Body string   `json:"body"`
	Tags []string `json:"tags,omitempty"`
	Urls []string `json:"urls",omitempty`
}

func Notify(msg string, config types.AppriseConfig) error {

	payload := Request{
		Body: msg,
		Urls: config.Urls,
		Tags: config.Tags,
	}

	jsonData, _ := json.Marshal(payload)

	url := config.Url + "/notify/"

	if config.Config != "" {
		url += config.Config
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}
