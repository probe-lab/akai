package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/probe-lab/akai/config"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const APIversion = "v1"

var ErrNetworkNotSupported = fmt.Errorf("network not supported")

type ServiceOption func(*Service) error

var DefaulServiceConfig = &config.AkaiAPIServiceConfig{
	Network:    config.DefaultNetwork.String(),
	Host:       "127.0.0.1",
	Port:       8080,
	PrefixPath: "api/" + APIversion,
	Timeout:    10 * time.Second,
	Mode:       "release",
	Meter:      otel.GetMeterProvider().Meter("akai_api_server"),
}

type Service struct {
	config *config.AkaiAPIServiceConfig

	appNewBlockHandler func(context.Context, Block) error
	appNewItemHandler  func(context.Context, DASItem) error
	appNewItemsHandler func(context.Context, []DASItem) error

	engine *gin.Engine
}

func NewService(cfg *config.AkaiAPIServiceConfig, opts ...ServiceOption) (*Service, error) {
	apiService := &Service{
		config:             cfg,
		appNewBlockHandler: func(context.Context, Block) error { return nil },
		appNewItemHandler:  func(context.Context, DASItem) error { return nil },
		appNewItemsHandler: func(context.Context, []DASItem) error { return nil },
	}

	apiPathBase := fmt.Sprintf("https://%s:%d/%s", cfg.Host, cfg.Port, cfg.PrefixPath)

	log.WithFields(log.Fields{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"timeout":  cfg.Timeout,
		"network":  cfg.Network,
		"api-path": apiPathBase,
	}).Info()
	return apiService, nil
}

func WithAppBlockHandler(blockHandler func(context.Context, Block) error) ServiceOption {
	return func(s *Service) error {
		s.appNewBlockHandler = blockHandler
		return nil
	}
}

func WithAppItemHandler(itemHandler func(context.Context, DASItem) error) ServiceOption {
	return func(s *Service) error {
		s.appNewItemHandler = itemHandler
		return nil
	}
}

func WithAppItemsHandler(itemsHandler func(context.Context, []DASItem) error) ServiceOption {
	return func(s *Service) error {
		s.appNewItemsHandler = itemsHandler
		return nil
	}
}

func (s *Service) Serve(ctx context.Context) error {
	return s.serve(ctx)
}

func (s *Service) serve(ctx context.Context) error {
	defer log.Info("closing Akai API service")

	gin.SetMode(s.config.Mode)
	s.engine = gin.New()
	s.engine.Use(gin.Recovery())

	// define basic endpoints
	s.engine.GET(s.config.PrefixPath+"/ping", s.pingHandler)
	s.engine.GET(s.config.PrefixPath+"/supported-network", s.getSupportedNetworkHandler)

	// Define app specific POST endpoints
	// Define a POST endpoint
	s.engine.POST(s.config.PrefixPath+"/new-block", s.postNewBlockHandler)
	s.engine.POST(s.config.PrefixPath+"/new-item", s.postNewItemHandler)
	s.engine.POST(s.config.PrefixPath+"/new-items", s.postNewItemsHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler: s.engine,
	}

	errC := make(chan error)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errC <- err
		} else {
			errC <- (error)(nil)
		}
	}()

	var err error
	select {
	case <-ctx.Done():
		srv.Shutdown(ctx)
		break
	case err = <-errC:
		break
	}
	return err
}

func (s *Service) isValidNetwork(remNet string) bool {
	return s.config.Network == remNet // we don't care about the NetworkID (its for the DB)
}

func (s *Service) pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, PongReply{Msg: "pong"})
}

func (s *Service) getSupportedNetworkHandler(c *gin.Context) {
	c.JSON(http.StatusOK, SupportedNetworks{Network: config.NetworkFromStr(s.config.Network)})
}

func (s *Service) UpdateNewBlockHandler(fn func(context.Context, Block) error) {
	s.appNewBlockHandler = fn
}

func (s *Service) UpdateNewItemHandler(fn func(context.Context, DASItem) error) {
	s.appNewItemHandler = fn
}

func (s *Service) UpdateNewItemsHandler(fn func(context.Context, []DASItem) error) {
	s.appNewItemsHandler = fn
}

func (s *Service) postNewBlockHandler(c *gin.Context) {
	hlog := log.WithFields(log.Fields{
		"service": "api-service",
		"op":      "POST new block",
	})
	hlog.Debug("new api-request")

	var block Block
	if err := c.BindJSON(&block); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	// ensure the block belongs to a supported network
	if !s.isValidNetwork(block.Network) {
		err := ErrNetworkNotSupported
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
	}

	// make app specific
	if err := s.appNewBlockHandler(c.Request.Context(), block); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	c.JSON(http.StatusOK, ACK{Status: "ok", Error: ""})
}

func (s *Service) postNewItemHandler(c *gin.Context) {
	hlog := log.WithFields(log.Fields{
		"service": "api-service",
		"op":      "POST new item",
	})
	hlog.Debug("new api-request")

	var item DASItem
	if err := c.BindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	// ensure the item belongs to a supported network
	if !s.isValidNetwork(item.Network) {
		err := ErrNetworkNotSupported
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	// assume that the item is correct
	// make app specific
	if err := s.appNewItemHandler(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
	}

	c.JSON(http.StatusOK, ACK{Status: "ok", Error: ""})
}

func (s *Service) postNewItemsHandler(c *gin.Context) {
	hlog := log.WithFields(log.Fields{
		"service": "api-service",
		"op":      "POST multiple new items",
	})
	hlog.Debug("new api-request")

	items := make([]DASItem, 0)
	if err := c.BindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	// assume that the item is correct
	// make app specific
	if err := s.appNewItemsHandler(c.Request.Context(), items); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	c.JSON(http.StatusOK, ACK{Status: "ok", Error: ""})
}
