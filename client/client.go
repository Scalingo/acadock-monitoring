package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Scalingo/acadock-monitoring/mem"
	"gopkg.in/errgo.v1"
)

type MemoryUsage mem.Usage

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

func (c *Client) CpuUsage(dockerId string) (int64, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/containers/"+dockerId+"/cpu", nil)
	if err != nil {
		return -1, errgo.Mask(err)
	}

	cpu, err := c.getInt(req)
	if err != nil {
		return -1, errgo.Mask(err)
	}

	return cpu, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Add("Content-Type", "text/plain")
	req.Header.Add("User-Agent", "Acadocker Client v1")
	return http.DefaultClient.Do(req)
}

func (c *Client) getInt(req *http.Request) (int64, error) {
	res, err := c.do(req)
	if err != nil {
		return -1, errgo.Mask(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return -1, errgo.Mask(err)
	}

	i, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return -1, errgo.Mask(err)
	}

	return i, nil
}
