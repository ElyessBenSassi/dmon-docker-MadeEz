package docker

import (
	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client.
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client using the default environment variables.
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

// Close releases any resources held by the underlying client.
func (c *Client) Close() error {
	return c.cli.Close()
}
