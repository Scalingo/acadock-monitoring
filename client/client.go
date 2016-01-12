package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Scalingo/go-netstat"
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

type NetUsage struct {
	netstat.NetworkStat
	RxBps int64 `json:"rx_bps"`
	TxBps int64 `json:"tx_bps"`
}

type Client struct {
	Endpoint string
}

type Usage struct {
	Memory *MemoryUsage `json:"memory"`
	Cpu    *CpuUsage    `json:"cpu"`
	Net    *NetUsage    `json:"net,omitempty"`
}

type ContainersUsage map[string]Usage

func NewContainersUsage() ContainersUsage {
	return ContainersUsage(make(map[string]Usage))
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
	var mem *MemoryUsage
	err := c.getResource(dockerId, "mem", mem)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	return mem, nil
}

func (c *Client) CpuUsage(dockerId string) (*CpuUsage, error) {
	var cpu *CpuUsage
	err := c.getResource(dockerId, "cpu", cpu)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	return cpu, nil
}

func (c *Client) NetUsage(dockerId string) (*NetUsage, error) {
	var net *NetUsage
	err := c.getResource(dockerId, "net", net)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	return net, nil
}

func (c *Client) Usage(dockerId string, net bool) (*Usage, error) {
	usage := &Usage{}
	err := c.getResourceWithQuery(dockerId, "usage", fmt.Sprintf("net=%v", net), usage)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	return usage, nil
}

func (c *Client) getResource(dockerId, resourceType string, data interface{}) error {
	return c.getResourceWithQuery(dockerId, resourceType, "", data)
}

func (c *Client) getResourceWithQuery(dockerId, resourceType string, query string, data interface{}) error {
	req, err := http.NewRequest("GET", c.Endpoint+"/containers/"+dockerId+"/"+resourceType+"?"+query, nil)
	if err != nil {
		return errgo.Mask(err)
	}

	res, err := c.do(req)
	if err != nil {
		return errgo.Mask(err)
	}

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return errgo.Mask(err)
	}

	return nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "Acadocker Client v1")
	return http.DefaultClient.Do(req)
}
