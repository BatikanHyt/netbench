package protocols

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/BatikanHyt/netbench/pkg/collector"
	"golang.org/x/net/http2"
)

type httpClient struct {
	Client      *http.Client
	Req         *http.Request
	ReportChan  chan *collector.HttpEntry
	readSize    int64
	writeSize   int64
	Url         string            `json:"url"`
	Method      string            `json:"method"`
	Version     string            `json:"version"`
	Body        string            `json:"body"`
	BodyFile    string            `json:"body_file"`
	Proxy       string            `json:"proxy"`
	Headers     map[string]string `json:"headers"`
	Timeout     time.Duration     `json:"Timeout"`
	Keep_alive  bool              `json:"keep-alive"`
	Compression bool              `json:"compression"`
	Redirect    bool              `json:"redirect"`
	initialized bool
	Auth        struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"auth"`
}

func NewHttpClient() *httpClient {
	client := &httpClient{
		Method:      "GET",
		Version:     "1.1",
		initialized: false,
	}
	return client
}

func (c *httpClient) Initialize(clc *collector.StatBase) {
	hclc, _ := (*clc).(*collector.HttpStatCollector)
	c.ReportChan = hclc.StatChannel
	tr := &http.Transport{
		DisableKeepAlives:  !c.Keep_alive,
		DisableCompression: !c.Compression,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return DialContextWithBytesTracked(ctx, network, address, &c.readSize, &c.writeSize)
		},
	}

	if c.Version == "2" {
		http2.ConfigureTransport(tr)
	}

	if c.Proxy != "" {
		proxyUrl, err := url.Parse(c.Proxy)
		if err == nil {
			fmt.Printf("Unable to set proxy %s", c.Proxy)
			return
		}
		tr.Proxy = http.ProxyURL(proxyUrl)
	}
	c.Client = &http.Client{
		Transport: tr,
		Timeout:   time.Second * c.Timeout}

	//Disable Redirect
	if !c.Redirect {
		c.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	fmt.Printf("Running HTTP bench for url %s\n", c.Url)
	c.initialized = true
}

func (c *httpClient) StartBenchmark() {
	if !c.initialized {
		fmt.Println("HTTP not initialized correctly!")
		return
	}

	c.makeRequest()
}

func (c *httpClient) makeRequest() {
	start := time.Now()
	req, rErr := c.createRequest()
	if rErr != nil {
		fmt.Printf("Error %s\n", rErr.Error())
		return
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
		elapsed := time.Since(start)
		stat := &collector.HttpEntry{
			ResponseCode: 1000,
			WriteSize:    c.writeSize,
			ReadSize:     c.readSize,
			Duration:     elapsed,
		}
		c.ReportChan <- stat
		return
	}
	defer resp.Body.Close()
	_, bErr := io.Copy(io.Discard, resp.Body)
	if bErr != nil {
		return
	}
	elapsed := time.Since(start)
	stat := &collector.HttpEntry{
		ResponseCode: resp.StatusCode,
		WriteSize:    c.writeSize,
		ReadSize:     c.readSize,
		Duration:     elapsed,
	}
	c.ReportChan <- stat
}

func (c *httpClient) createRequest() (*http.Request, error) {
	var dataReader io.Reader
	var err error
	var req *http.Request

	if c.BodyFile != "" {
		var content []byte
		content, err = os.ReadFile(c.BodyFile)
		if err != nil {
			return nil, err
		}
		dataReader = bytes.NewReader(content)
	} else {
		dataReader = strings.NewReader(c.Body)
	}
	req, err = http.NewRequest(c.Method, c.Url, io.NopCloser(dataReader))
	if err != nil {
		return req, err
	}
	if c.Auth.Username != "" && c.Auth.Password != "" {
		req.SetBasicAuth(c.Auth.Username, c.Auth.Password)
	}
	for key, value := range c.Headers {
		if strings.EqualFold(key, "host") {
			req.Host = value
		} else {
			req.Header.Set(key, value)
		}
	}
	return req, err
}
