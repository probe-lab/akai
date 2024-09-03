package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

type ServiceOption func(*Service) error

const APIversion = "v1"

var DefaulServiceConfig = config.AkaiAPIServiceConfig{
	Network:    config.DefaultNetwork.String(),
	Host:       "127.0.0.1",
	Port:       8080,
	PrefixPath: "api/" + APIversion,
	Timeout:    10 * time.Second,
	Mode:       "release",
}

type Service struct {
	config config.AkaiAPIServiceConfig

	appNewBlobHandler     func(context.Context, Blob) error
	appNewSegmentHandler  func(context.Context, BlobSegment) error
	appNewSegmentsHandler func(context.Context, []BlobSegment) error

	engine *gin.Engine
}

func NewService(cfg config.AkaiAPIServiceConfig, opts ...ServiceOption) (*Service, error) {
	apiService := &Service{
		config:                cfg,
		appNewBlobHandler:     func(context.Context, Blob) error { return nil },
		appNewSegmentHandler:  func(context.Context, BlobSegment) error { return nil },
		appNewSegmentsHandler: func(context.Context, []BlobSegment) error { return nil },
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

func WithAppBlobHandler(blobHandler func(context.Context, Blob) error) ServiceOption {
	return func(s *Service) error {
		s.appNewBlobHandler = blobHandler
		return nil
	}
}

func WithAppSegmentHandler(segmentHandler func(context.Context, BlobSegment) error) ServiceOption {
	return func(s *Service) error {
		s.appNewSegmentHandler = segmentHandler
		return nil
	}
}

func WithAppSegmentsHandler(segmentsHandler func(context.Context, []BlobSegment) error) ServiceOption {
	return func(s *Service) error {
		s.appNewSegmentsHandler = segmentsHandler
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
	s.engine.POST(s.config.PrefixPath+"/new-blob", s.postNewBlobHandler)
	s.engine.POST(s.config.PrefixPath+"/new-segment", s.postNewSegmentHandler)
	s.engine.POST(s.config.PrefixPath+"/new-segments", s.postNewSegmentsHandler)

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

	select {
	case <-ctx.Done():
		return nil
	case err := <-errC:
		return err
	}
}

func (s *Service) isNetworkSupported(remNet models.Network) bool {
	return s.config.Network == remNet.String() // we don't care about the NetworkID (its for the DB)
}

func (s *Service) pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, PongReply{Msg: "pong"})
}

func (s *Service) getSupportedNetworkHandler(c *gin.Context) {
	c.JSON(http.StatusOK, SupportedNetworks{Network: models.NetworkFromStr(s.config.Network)})
}

func (s *Service) UpdateNewBlobHandler(fn func(context.Context, Blob) error) {
	s.appNewBlobHandler = fn
}

func (s *Service) UpdateNewSegmentHandler(fn func(context.Context, BlobSegment) error) {
	s.appNewSegmentHandler = fn
}

func (s *Service) UpdateNewSegmentsHandler(fn func(context.Context, []BlobSegment) error) {
	s.appNewSegmentsHandler = fn
}

func (s *Service) postNewBlobHandler(c *gin.Context) {
	hlog := log.WithFields(log.Fields{
		"service": "api-service",
		"op":      "POST new blob",
	})
	hlog.Debug("new api-request")

	var blob Blob
	if err := c.BindJSON(&blob); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	// ensure the block belongs to a supported network
	if !s.isNetworkSupported(blob.Network) {
		err := ErrNetworkNotSupported
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
	}

	// make app specific
	if err := s.appNewBlobHandler(c.Request.Context(), blob); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
	}

	c.JSON(http.StatusOK, ACK{Status: "ok", Error: ""})
}

func (s *Service) postNewSegmentHandler(c *gin.Context) {
	hlog := log.WithFields(log.Fields{
		"service": "api-service",
		"op":      "POST new segment",
	})
	hlog.Debug("new api-request")

	var segment BlobSegment
	if err := c.BindJSON(&segment); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	// assume that the segment is correct

	// make app specific
	if err := s.appNewSegmentHandler(c.Request.Context(), segment); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
	}

	c.JSON(http.StatusOK, ACK{Status: "ok", Error: ""})
}

func (s *Service) postNewSegmentsHandler(c *gin.Context) {
	hlog := log.WithFields(log.Fields{
		"service": "api-service",
		"op":      "POST multiple new segments",
	})
	hlog.Debug("new api-request")

	segments := make([]BlobSegment, 0)
	if err := c.BindJSON(&segments); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	// assume that the segment is correct
	// make app specific
	if err := s.appNewSegmentsHandler(c.Request.Context(), segments); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
	}

	c.JSON(http.StatusOK, ACK{Status: "ok", Error: ""})
}
