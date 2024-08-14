package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	log "github.com/sirupsen/logrus"
)

const APIversion = "v1"

type ServiceConfig struct {
	// api host parameters
	Host       string
	Port       int
	PrefixPath string
	Timeout    time.Duration

	// configuration
	Network               db.Network
	appNewBlobHandler     func(Blob) error
	appNewSegmentHandler  func(BlobSegment) error
	appNewSegmentsHandler func([]BlobSegment) error
}

var DefaulServiceConfig = ServiceConfig{
	Network:               config.DefaultNetwork,
	Host:                  "127.0.0.1",
	Port:                  8080,
	PrefixPath:            "api/" + APIversion,
	Timeout:               10 * time.Second,
	appNewBlobHandler:     func(b Blob) error { return nil },
	appNewSegmentHandler:  func(bs BlobSegment) error { return nil },
	appNewSegmentsHandler: func(bs []BlobSegment) error { return nil },
}

var DefaultServiceConfig ServiceConfig = ServiceConfig{}

type Service struct {
	config ServiceConfig

	router *gin.Engine
}

func NewService(cfg ServiceConfig) (*Service, error) {
	apiService := &Service{
		config: cfg,
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

func (s *Service) Serve(mode string) error {
	gin.SetMode(mode)
	s.router = gin.New()
	s.router.Use(gin.Recovery())

	// define basic endpoints
	s.router.GET(s.config.PrefixPath+"/ping", s.pingHandler)
	s.router.GET(s.config.PrefixPath+"/supported-network", s.getSupportedNetworkHandler)

	// Define app specific POST endpoints
	// Define a POST endpoint
	s.router.POST(s.config.PrefixPath+"/new-blob", s.postNewBlobHandler)
	s.router.POST(s.config.PrefixPath+"/new-segment", s.postNewSegmentHandler)
	s.router.POST(s.config.PrefixPath+"/new-segments", s.postNewSegmentsHandler)

	// Start the server on port 8080
	return s.router.Run(fmt.Sprintf("%s:%d", s.config.Host, s.config.Port))
}

func (s *Service) isNetworkSupported(remNet db.Network) bool {
	return s.config.Network.String() == remNet.String() // we don't care about the NetworkID (its for the DB)
}

func (s *Service) pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, PongReply{Msg: "pong"})
}

func (s *Service) getSupportedNetworkHandler(c *gin.Context) {
	c.JSON(http.StatusOK, SupportedNetworks{Network: s.config.Network})
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
	if err := s.config.appNewBlobHandler(blob); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
	}

	// if the block already includes Segments, process them
	if blob.BlobSegments.Segments != nil {
		if err := s.config.appNewSegmentsHandler(blob.BlobSegments.Segments); err != nil {
			c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
			hlog.Error(err)
		}
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
	if err := s.config.appNewSegmentHandler(segment); err != nil {
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

	segments := BlobSegments{
		Segments: make([]BlobSegment, 0),
	}
	if err := c.BindJSON(&segments); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
		return
	}

	// assume that the segment is correct
	// make app specific
	if err := s.config.appNewSegmentsHandler(segments.Segments); err != nil {
		c.JSON(http.StatusBadRequest, ACK{Status: "error", Error: err.Error()})
		hlog.Error(err)
	}

	c.JSON(http.StatusOK, ACK{Status: "ok", Error: ""})
}
