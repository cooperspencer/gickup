package apprise

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cooperspencer/gickup/types"
)

type Request struct {
	Body string   `json:"body"`
	Tags []string `json:"tags,omitempty"`
	Urls []string `json:"urls",omitempty`
}

type ErrorMsg struct {
	Error string `json:"error"`
}

func Notify(msg string, config types.AppriseConfig) error {

	payload := Request{
		Body: msg,
		Urls: config.Urls,
		Tags: config.Tags,
	}

	jsonData, _ := json.Marshal(payload)

	if !strings.HasSuffix(config.Url, "/") {
		config.Url += "/"
	}

	url := config.Url + "notify/"

	if config.Config != "" {
		url += config.Config
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	errormsg := ErrorMsg{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &errormsg)
	if err != nil {
		return err
	}

	if errormsg.Error != "" {
		return errors.New(errormsg.Error)
	}

	return nil
}
