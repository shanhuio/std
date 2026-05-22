package docker

// Ping checks that the Docker daemon is reachable.
func Ping(c *Client) error {
	resp, err := c.get("_ping", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
