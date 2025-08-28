package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Scalingo/go-netstat"
	"github.com/Scalingo/go-utils/errors/v3"
)

var _ AcadockClient = &Client{}

type MemoryUsage struct {
	MemoryUsage    uint64 `json:"memory_usage"`
	SwapUsage      uint64 `json:"swap_usage"`
	MemoryLimit    uint64 `json:"memory_limit"`
	SwapLimit      uint64 `json:"swap_limit"`
	MaxMemoryUsage uint64 `json:"max_memory_usage"`
	MaxSwapUsage   uint64 `json:"max_swap_usage"`
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
	MemoryCommitted uint64 `json:"memory_committed"`
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
	AllContainersUsage(ctx context.Context) (ContainersUsage, error)
	Memory(ctx context.Context, dockerId string) (*MemoryUsage, error)
	CpuUsage(ctx context.Context, dockerId string) (*CpuUsage, error)
	NetUsage(ctx context.Context, dockerId string) (*NetUsage, error)
	Usage(ctx context.Context, dockerId string, net bool) (*Usage, error)
	HostUsage(ctx context.Context, opts HostUsageOpts) (HostUsage, error)
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

func NewClient(ctx context.Context, endpoint string, opts ...ClientOpts) (*Client, error) {
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "parse endpoint '%s'", endpoint)
	}

	c := &Client{Endpoint: endpoint}
	for _, opt := range opts {
		c = opt(c)
	}
	return c, nil
}

func (c *Client) Memory(ctx context.Context, dockerId string) (*MemoryUsage, error) {
	var mem *MemoryUsage
	err := c.getResource(ctx, dockerId, "mem", mem)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get container memory usage")
	}
	return mem, nil
}

func (c *Client) CpuUsage(ctx context.Context, dockerId string) (*CpuUsage, error) {
	var cpu *CpuUsage
	err := c.getResource(ctx, dockerId, "cpu", cpu)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get container cpu usage")
	}
	return cpu, nil
}

func (c *Client) NetUsage(ctx context.Context, dockerId string) (*NetUsage, error) {
	var net *NetUsage
	err := c.getResource(ctx, dockerId, "net", net)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get container network usage")
	}

	return net, nil
}

func (c *Client) Usage(ctx context.Context, dockerId string, net bool) (*Usage, error) {
	usage := &Usage{}
	err := c.getResourceWithQuery(ctx, dockerId, "usage", fmt.Sprintf("net=%v", net), usage)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get container usage")
	}
	return usage, nil
}

func (c *Client) AllContainersUsage(ctx context.Context) (ContainersUsage, error) {
	var usage ContainersUsage
	err := c.getResource(ctx, "", "usage", &usage)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get all containers usage")
	}
	return usage, nil
}

type HostUsageOpts struct {
	IncludeContainerIfLabel string
}

func (c *Client) HostUsage(ctx context.Context, opts HostUsageOpts) (HostUsage, error) {
	query := ""
	if opts.IncludeContainerIfLabel != "" {
		query = fmt.Sprintf("include_container_if_label=%s", opts.IncludeContainerIfLabel)
	}
	var res HostUsage
	err := c.getPathWithQuery(ctx, "/host/usage", query, &res)
	if err != nil {
		return res, errors.Wrap(ctx, err, "get host usage")
	}
	return res, nil
}

func (c *Client) getResource(ctx context.Context, dockerId, resourceType string, data interface{}) error {
	return c.getResourceWithQuery(ctx, dockerId, resourceType, "", data)
}

func (c *Client) getResourceWithQuery(ctx context.Context, dockerId, resourceType string, query string, data interface{}) error {
	var path string
	if dockerId == "" {
		path = "/containers/" + resourceType
	} else {
		path = "/containers/" + dockerId + "/" + resourceType
	}
	return c.getPathWithQuery(ctx, path, query, data)
}

func (c *Client) getPathWithQuery(ctx context.Context, path, query string, data interface{}) error {
	endpoint := c.Endpoint + path

	if query != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return errors.Wrap(ctx, err, "create new http request")
	}

	res, err := c.do(req)
	if err != nil {
		return errors.Wrap(ctx, err, "execute http request")
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return errors.Wrap(ctx, err, "decode response body payload")
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
