package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/probe-lab/akai/config"
	log "github.com/sirupsen/logrus"
)

var DefaultClientConfig = &config.AkaiAPIClientConfig{
	Host:       "127.0.0.1",
	Port:       8080,
	PrefixPath: "api/" + APIversion,
	Timeout:    10 * time.Second,
}

type Client struct {
	config *config.AkaiAPIClientConfig

	base    *url.URL
	address string
	client  *http.Client
}

func NewClient(cfg *config.AkaiAPIClientConfig) (*Client, error) {
	// http client for the communication
	httpCli := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   cfg.Timeout,
				KeepAlive: 40 * time.Second,
			}).DialContext,
		},
	}

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	urlBase, err := url.Parse(fmt.Sprintf("http://%s/%s", address, cfg.PrefixPath))
	if err != nil {
		return nil, errors.Wrap(err, "composing Akai API Client's base URL")
	}

	apiCli := &Client{
		config:  cfg,
		base:    urlBase,
		address: address,
		client:  httpCli,
	}

	// test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	err = apiCli.CheckConnection(ctx)
	if err != nil {
		log.Error("not connections with the akai-api-server", err)
		return nil, err
	}

	log.WithFields(log.Fields{
		"host":   cfg.Host,
		"port":   cfg.Port,
		"prefix": cfg.PrefixPath,
	}).Info("successful conenction to akai-api-server")

	return apiCli, nil
}

func (c *Client) Serve(ctx context.Context) error {
	<-ctx.Done()
	c.client.CloseIdleConnections()
	return nil
}

func (c *Client) CheckConnection(ctx context.Context) error {
	// try the API agains the V2Version call
	if err := c.Ping(ctx); err != nil {
		return errors.Wrap(err, "testing connectivity")
	}
	return nil
}

func (c *Client) get(
	ctx context.Context,
	endpoint string,
	query string,
) ([]byte, error) {
	var respBody []byte

	// set deadline
	opCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	callURL := composeCallURL(c.base, endpoint, query)

	req, err := http.NewRequestWithContext(opCtx, http.MethodGet, callURL.String(), nil)
	if err != nil {
		return []byte{}, errors.Wrap(err, "unable to compose call URL")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return respBody, errors.Wrap(err, fmt.Sprintf("unable to request URL %s", callURL.String()))
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return respBody, errors.Wrap(err, "reading response body")
	}

	// return copy of the request
	return respBody, nil
}

func (c *Client) post(
	ctx context.Context,
	endpoint string,
	item any,
) ([]byte, error) {
	var respBody []byte

	// set deadline
	opCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// compose body
	bodyBytes, err := json.Marshal(item)
	if err != nil {
		return []byte{}, errors.Wrap(err, "marshalling body for POST API request")
	}
	body := bytes.NewReader(bodyBytes)

	// compose url
	callURL := composeCallURL(c.base, endpoint, "")

	// create the request
	req, err := http.NewRequestWithContext(opCtx, http.MethodPost, callURL.String(), body)
	if err != nil {
		return []byte{}, errors.Wrap(err, "unable to compose call URL")
	}
	req.Header.Set("Accept", "application/json")

	// make the query
	resp, err := c.client.Do(req)
	if err != nil {
		return respBody, errors.Wrap(err, fmt.Sprintf("unable to request URL %s", callURL.String()))
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return respBody, errors.Wrap(err, "reading response body")
	}

	// return copy of the request
	return respBody, nil
}

func composeCallURL(base *url.URL, endpoint, query string) *url.URL {
	callURL := *base
	callURL.Path += endpoint
	if callURL.RawQuery == "" {
		callURL.RawQuery = query
	} else if query != "" {
		callURL.RawQuery = fmt.Sprintf("%s&%s", callURL.RawQuery, query)
	}

	return &callURL
}

func (c *Client) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	// make Http request
	resp, err := c.get(ctx, "/ping", "")
	if err != nil {
		return errors.Wrap(err, "sending ping to API server")
	}

	// parse response into V2Version
	var pong PongReply
	err = json.Unmarshal(resp, &pong)
	if err != nil {
		return errors.Wrap(err, "unmarshaling pong message from http request")
	}
	return nil
}

func (c *Client) GetSupportedNetworks(ctx context.Context) (SupportedNetworks, error) {
	var supNetwork SupportedNetworks
	// make Http request
	resp, err := c.get(ctx, "/supported-network", "")
	if err != nil {
		return supNetwork, errors.Wrap(err, "sending ping to API server")
	}

	// parse response into V2Version
	err = json.Unmarshal(resp, &supNetwork)
	if err != nil {
		return supNetwork, errors.Wrap(err, "unmarshaling supported network message from http request")
	}
	return supNetwork, nil
}

func (c *Client) PostNewBlob(ctx context.Context, blob Blob) error {
	// make Http request
	resp, err := c.post(ctx, "/new-blob", blob)
	if err != nil {
		return errors.Wrap(err, "sending new blob to API server")
	}

	// parse response into V2Version
	var ack ACK
	err = json.Unmarshal(resp, &ack)
	if err != nil {
		return errors.Wrap(err, "error unmarshaling API's response")
	}
	if ack.Status != "ok" {
		return fmt.Errorf(ack.Error)
	}
	return nil
}

func (c *Client) PostNewSegment(ctx context.Context, segment BlobSegment) error {
	// make Http request
	resp, err := c.post(ctx, "/new-segment", segment)
	if err != nil {
		return errors.Wrap(err, "sending new segment to API server")
	}

	// parse response into V2Version
	var ack ACK
	err = json.Unmarshal(resp, &ack)
	if err != nil {
		return errors.Wrap(err, "error unmarshaling API's response")
	}
	if ack.Status != "ok" {
		return fmt.Errorf(ack.Error)
	}
	return nil
}

func (c *Client) PostNewSegments(ctx context.Context, segments []BlobSegment) error {
	// make Http request
	resp, err := c.post(ctx, "/new-segments", segments)
	if err != nil {
		return errors.Wrap(err, "sending new segments to API server")
	}

	// parse response into V2Version
	var ack ACK
	err = json.Unmarshal(resp, &ack)
	if err != nil {
		return errors.Wrap(err, "error unmarshaling API's response")
	}
	if ack.Status != "ok" {
		return fmt.Errorf(ack.Error)
	}
	return nil
}
