package docker

// Ping checks that the Docker daemon is reachable.
func (c *Client) Ping() error {
	resp, err := c.get("_ping", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
