package api

import (
	"context"
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

var DefaultClientConfig = config.AvailAPIConfig{
	Host:    "localhost",
	Port:    5000,
	Timeout: 10 * time.Second,
}

// coming from Avail's API website
// https://docs.availproject.org/docs/operate-a-node/run-a-light-client/light-client-api-reference

type HTTPClient struct {
	base    *url.URL
	address string
	client  *http.Client
	timeout time.Duration
}

func NewHTTPCli(opts config.AvailAPIConfig) (*HTTPClient, error) {

	// http client for the communication
	httpCli := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   opts.Timeout,
				KeepAlive: 40 * time.Second,
			}).DialContext,
			IdleConnTimeout: 600 * time.Second,
		},
	}

	address := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	urlBase, err := url.Parse(fmt.Sprintf("http://%s", address))
	if err != nil {
		return nil, errors.Wrap(err, "composing Avail API's base URL")
	}

	cli := &HTTPClient{
		base:    urlBase,
		address: address,
		client:  httpCli,
		timeout: opts.Timeout,
	}

	return cli, nil
}

func (c *HTTPClient) Serve(ctx context.Context) error {

	err := c.CheckConnection(ctx)
	if err != nil {
		log.Error("not connections with the avail-light-client", err)
		return err
	}

	<-ctx.Done()

	c.client.CloseIdleConnections()

	return nil
}

func (c *HTTPClient) CheckConnection(ctx context.Context) error {
	// try the API agains the V2Version call

	version, err := c.GetV2Version(ctx)
	if err != nil {
		return errors.Wrap(err, "testing connectivity")
	}

	log.WithFields(log.Fields{
		"cli_version":     version.Version,
		"network-version": version.NetworkVersion,
	}).Info("successfull connection to trusted Avail Node")

	return nil
}

func (c *HTTPClient) get(
	ctx context.Context,
	endpoint string,
	query string) ([]byte, error) {

	var respBody []byte

	// set deadline
	opCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	callURL := composeCallURL(c.base, endpoint, query)
	fmt.Printf("%s\n", callURL)

	req, err := http.NewRequestWithContext(opCtx, http.MethodGet, callURL.String(), nil)
	if err != nil {
		return []byte{}, errors.Wrap(err, "unable to compose call URL")
	}

	// we will only handle JSONs
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

func (c *HTTPClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}
