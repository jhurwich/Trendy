// Copyright 2015 Jordan Hurwich - no license granted

package markit

import (
	"errors"
	"io"
	"net/http"
	"os"
)

func PollNewData(symbol string) (string, error) {
	resp, err := http.Get("http://google.com")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	return "", nil
}
