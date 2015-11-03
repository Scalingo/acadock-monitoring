package client

import (
	"encoding/json"
	"net/http"
	"net/url"

	"gopkg.in/errgo.v1"
)

type MemoryUsage struct {
	MemoryUsage    int64 `json:"memory_usage"`
	SwapUsage      int64 `json:"swap_usage"`
	MemoryLimit    int64 `json:"memory_limit"`
	SwapLimit      int64 `json:"swap_limit"`
	MaxMemoryUsage int64 `json:"max_memory_usage"`
	MaxSwapUsage   int64 `json:"max_swap_usage"`
}

type CpuUsage struct {
	UsageInPercents int `json:"usage_in_percents"`
}

type Client struct {
	Endpoint string
}

func NewClient(endpoint string) (*Client, error) {
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	c := &Client{Endpoint: endpoint}
	return c, nil
}

func (c *Client) Memory(dockerId string) (*MemoryUsage, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/containers/"+dockerId+"/mem", nil)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	res, err := c.do(req)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	defer res.Body.Close()

	var mem *MemoryUsage
	err = json.NewDecoder(res.Body).Decode(&mem)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	return mem, nil
}

func (c *Client) CpuUsage(dockerId string) (*CpuUsage, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/containers/"+dockerId+"/cpu", nil)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	res, err := c.do(req)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	defer res.Body.Close()

	var cpu *CpuUsage
	err = json.NewDecoder(res.Body).Decode(&cpu)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	return cpu, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "Acadocker Client v1")
	return http.DefaultClient.Do(req)
}
