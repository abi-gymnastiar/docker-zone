package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
)

type Client struct {
	http *http.Client
}

type container struct {
	ID    string `json:"Id"`
	State struct {
		Status  string `json:"Status"`
		Running bool   `json:"Running"`
	} `json:"State"`
}

type Status struct {
	ID      string
	State   string
	Running bool
}

func NewClient(socket string) *Client {
	return &Client{http: &http.Client{Transport: &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socket)
		},
	}}}
}

func (c *Client) Status(name string) (Status, error) {
	req, err := http.NewRequest(http.MethodGet, "http://docker/containers/"+name+"/json", nil)
	if err != nil {
		return Status{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Status{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Status{State: "not_found"}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Status{}, fmt.Errorf("container %q returned HTTP %d", name, resp.StatusCode)
	}

	var result container
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Status{}, fmt.Errorf("decode container status: %w", err)
	}
	return Status{ID: result.ID, State: result.State.Status, Running: result.State.Running}, nil
}

func (c *Client) Proxy(method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://docker"+endpoint, body)
	if err != nil {
		return nil, err
	}
	return c.http.Do(req)
}
