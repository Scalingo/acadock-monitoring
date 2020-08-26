package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Scalingo/go-netstat"
	"gopkg.in/errgo.v1"
)

var _ AcadockClient = &Client{}

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

type HostUsage struct {
	CPU    HostCpuUsage    `json:"cpu"`
	Memory HostMemoryUsage `json:"memory"`
}
type HostCpuUsage struct {
	Usage                            float64 `json:"usage"`
	Amount                           int     `json:"amount"`
	QueueLengthExponentiallySmoothed float64 `json:"queue_length_exponentially_smoothed"`
}

type HostMemoryUsage struct {
	Free            uint64 `json:"free"`
	Total           uint64 `json:"total"`
	Swap            uint64 `json:"swap"`
	MemoryUsage     uint64 `json:"memory_usage"`
	MemoryCommitted uint64 `json:"memory_committed""`
	MaxMemoryUsage  uint64 `json:"max_memory_usage"`
	SwapUsage       uint64 `json:"swap_usage"`
	SwapCommitted   uint64 `json:"swap_committed"`
	MaxSwapUsage    uint64 `json:"max_swap_usage"`
}

type NetUsage struct {
	netstat.NetworkStat
	RxBps int64 `json:"rx_bps"`
	TxBps int64 `json:"tx_bps"`
}

type AcadockClient interface {
	AllContainersUsage() (ContainersUsage, error)
	Memory(dockerId string) (*MemoryUsage, error)
	CpuUsage(dockerId string) (*CpuUsage, error)
	NetUsage(dockerId string) (*NetUsage, error)
	Usage(dockerId string, net bool) (*Usage, error)
	HostUsage(opts HostUsageOpts) (HostUsage, error)
}

type Client struct {
	Endpoint string
	Username string
	Password string
}

type ClientOpts func(*Client) *Client

type Usage struct {
	Memory *MemoryUsage      `json:"memory"`
	Cpu    *CpuUsage         `json:"cpu"`
	Net    *NetUsage         `json:"net,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

type ContainersUsage map[string]Usage

func NewContainersUsage() ContainersUsage {
	return ContainersUsage(make(map[string]Usage))
}

func WithAuthentication(user, pass string) ClientOpts {
	return func(c *Client) *Client {
		c.Username = user
		c.Password = pass
		return c
	}
}

func NewClient(endpoint string, opts ...ClientOpts) (*Client, error) {
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	c := &Client{Endpoint: endpoint}
	for _, opt := range opts {
		c = opt(c)
	}
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

func (c *Client) AllContainersUsage() (ContainersUsage, error) {
	var usage ContainersUsage
	err := c.getResource("", "usage", &usage)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	return usage, nil
}

type HostUsageOpts struct {
	IncludeContainerIfLabel string
}

func (c *Client) HostUsage(opts HostUsageOpts) (HostUsage, error) {
	query := ""
	if opts.IncludeContainerIfLabel != "" {
		query = fmt.Sprintf("include_container_if_label=%s", opts.IncludeContainerIfLabel)
	}
	var res HostUsage
	err := c.getPathWithQuery("/host/usage", query, &res)
	if err != nil {
		return res, errgo.Notef(err, "fail to get host usage")
	}
	return res, nil
}

func (c *Client) getResource(dockerId, resourceType string, data interface{}) error {
	return c.getResourceWithQuery(dockerId, resourceType, "", data)
}

func (c *Client) getResourceWithQuery(dockerId, resourceType string, query string, data interface{}) error {
	var path string
	if dockerId == "" {
		path = "/containers/" + resourceType
	} else {
		path = "/containers/" + dockerId + "/" + resourceType
	}
	return c.getPathWithQuery(path, query, data)
}

func (c *Client) getPathWithQuery(path, query string, data interface{}) error {
	endpoint := c.Endpoint + path

	if query != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
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

	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	return http.DefaultClient.Do(req)
}
