package avail

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// coming from Avail's API website
// https://docs.availproject.org/docs/operate-a-node/run-a-light-client/light-client-api-reference

type HttpClientConfig struct {
	IP      string
	Port    int
	Timeout time.Duration
}

type HttpClient struct {
	ctx     context.Context
	base    *url.URL
	address string
	client  *http.Client
	timeout time.Duration
}

func New(ctx context.Context, opts HttpClientConfig) (*HttpClient, error) {

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

	address := fmt.Sprintf("%s:%d", opts.IP, opts.Port)
	urlBase, err := url.Parse(fmt.Sprintf("http://%s", address))
	if err != nil {
		return nil, errors.Wrap(err, "composing Avail API's base URL")
	}

	cli := &HttpClient{
		ctx:     ctx,
		base:    urlBase,
		address: address,
		client:  httpCli,
		timeout: opts.Timeout,
	}

	return cli, nil
}

func (c *HttpClient) CheckConnection() error {
	// try the API agains the V2Version call

	version, err := c.GetV2Version(c.ctx)
	if err != nil {
		return errors.Wrap(err, "testing connectivity")
	}

	log.WithFields(log.Fields{
		"cli_version":     version.Version,
		"network-version": version.NetworkVersion,
	}).Info("successfull connection to trusted Avail Node")

	return nil
}

func (c *HttpClient) get(
	ctx context.Context,
	endpoint string,
	query string) ([]byte, error) {

	var respBody []byte

	// set deadline
	opCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	callUrl := composeCallUrl(c.base, endpoint, query)

	req, err := http.NewRequestWithContext(opCtx, http.MethodGet, callUrl.String(), nil)
	if err != nil {
		return []byte{}, errors.Wrap(err, "unable to compose call URL")
	}

	// we will only handle JSONs
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return respBody, errors.Wrap(err, fmt.Sprintf("unable to request URL %s", callUrl.String()))
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return respBody, errors.Wrap(err, "reading response body")
	}

	// return copy of the request
	return respBody, nil
}

func composeCallUrl(base *url.URL, endpoint, query string) *url.URL {
	callUrl := *base
	callUrl.Path += endpoint
	if callUrl.RawQuery == "" {
		callUrl.RawQuery = query
	} else if query != "" {
		callUrl.RawQuery = fmt.Sprintf("%s&%s", callUrl.RawQuery, query)
	}

	return &callUrl
}
