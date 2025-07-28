package main

import (
	"io"
	"net/http"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

func getPublicIP() error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "IP-Address"})

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://ifconfig.me", nil)
	if err != nil {
		return err
	}
	req.Header.Add("User-Agent", "curl/7.54.1")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	t.AppendRow(table.Row{"1", string(bodyBytes)})
	t.Render()
	return nil
}
