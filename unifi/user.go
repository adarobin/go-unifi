package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-version"
)

var (
	controllerV6_0_43 = version.Must(version.NewVersion("6.0.43"))
)

// GetUserByMAC returns slightly different information than GetUser, as they
// use separate endpoints for their lookups. Specifically IP is only returned
// by this method.
func (c *Client) GetUserByMAC(ctx context.Context, site, mac string) (*User, error) {
	var respBody struct {
		Meta meta   `json:"meta"`
		Data []User `json:"data"`
	}

	err := c.do(ctx, "GET", fmt.Sprintf("s/%s/stat/user/%s", site, mac), nil, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *Client) CreateUser(ctx context.Context, site string, d *User) (*User, error) {
	reqBody := struct {
		Objects []struct {
			Data *User `json:"data"`
		} `json:"objects"`
	}{
		Objects: []struct {
			Data *User `json:"data"`
		}{
			{Data: d},
		},
	}

	var respBody struct {
		Meta meta `json:"meta"`
		Data []struct {
			Meta meta   `json:"meta"`
			Data []User `json:"data"`
		} `json:"data"`
	}

	err := c.do(ctx, "POST", fmt.Sprintf("s/%s/group/user", site), reqBody, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, fmt.Errorf("malformed group response")
	}

	if err := respBody.Data[0].Meta.error(); err != nil {
		return nil, err
	}

	if len(respBody.Data[0].Data) != 1 {
		return nil, &NotFoundError{}
	}

	new := respBody.Data[0].Data[0]

	return &new, nil
}

func (c *Client) stamgr(ctx context.Context, site, cmd string, data map[string]interface{}) ([]User, error) {
	reqBody := map[string]interface{}{}

	for k, v := range data {
		reqBody[k] = v
	}

	reqBody["cmd"] = cmd

	var respBody struct {
		Meta meta   `json:"meta"`
		Data []User `json:"data"`
	}

	err := c.do(ctx, "POST", fmt.Sprintf("s/%s/cmd/stamgr", site), reqBody, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody.Data, nil
}

func (c *Client) BlockUserByMAC(ctx context.Context, site, mac string) error {
	users, err := c.stamgr(ctx, site, "block-sta", map[string]interface{}{
		"mac": mac,
	})
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return &NotFoundError{}
	}
	return nil
}

func (c *Client) UnblockUserByMAC(ctx context.Context, site, mac string) error {
	users, err := c.stamgr(ctx, site, "unblock-sta", map[string]interface{}{
		"mac": mac,
	})
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return &NotFoundError{}
	}
	return nil
}

func (c *Client) DeleteUserByMAC(ctx context.Context, site, mac string) error {
	users, err := c.stamgr(ctx, site, "forget-sta", map[string]interface{}{
		"macs": []string{mac},
	})
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return &NotFoundError{}
	}
	return nil
}

func (c *Client) ListUser(ctx context.Context, site string) ([]User, error) {
	return c.listUser(ctx, site)
}

// GetUser returns information about a user from the REST endpoint.
// The GetUserByMAC method returns slightly different information (for
// example the IP) as it uses a different endpoint.
func (c *Client) GetUser(ctx context.Context, site, id string) (*User, error) {
	return c.getUser(ctx, site, id)
}

func (c *Client) UpdateUser(ctx context.Context, site string, d *User) (*User, error) {
	switch v, _ := version.NewVersion(c.version); {
	case v.GreaterThanOrEqual(controllerV6_0_43):
		return c.updateUser6_0_43(ctx, site, d)
	default:
		return c.updateUser(ctx, site, d)
	}
}

func (c *Client) updateUser6_0_43(ctx context.Context, site string, d *User) (*User, error) {
	var respBody struct {
		Meta meta   `json:"meta"`
		Data []User `json:"data"`
	}

	err := c.do(ctx, "PUT", fmt.Sprintf("s/%s/rest/user/%s", site, d.ID), d, &respBody)
	if err != nil {
		return nil, err
	}

	user, err := c.GetUser(ctx, site, d.ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
