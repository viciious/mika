package tracker

import (
	"bytes"
	"crypto/tls"
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
	"github.com/viciious/mika/config"
	"github.com/viciious/mika/consts"
	"github.com/viciious/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/toorop/gin-logrus"
	"net"
	"net/http"
	"strings"
	"time"
)

type errCode int

const (
	msgInvalidReqType       errCode = 100
	msgMissingInfoHash      errCode = 101
	msgMissingPeerID        errCode = 102
	msgMissingPort          errCode = 103
	msgInvalidPort          errCode = 104
	msgInvalidInfoHash      errCode = 150
	msgInvalidPeerID        errCode = 151
	msgInvalidNumWant       errCode = 152
	msgBadClient            errCode = 153
	msgOk                   errCode = 200
	msgInfoHashNotFound     errCode = 480
	msgInvalidAuth          errCode = 490
	msgClientRequestTooFast errCode = 500
	msgGenericError         errCode = 900
	msgMalformedRequest     errCode = 901
	msgQueryParseFail       errCode = 902
)

var (
	// Error code to message mappings
	responseStringMap = map[errCode]error{
		msgInvalidReqType:       errors.New("Invalid request type"),
		msgMissingInfoHash:      errors.New("info_hash missing from request"),
		msgMissingPeerID:        errors.New("peer_id missing from request"),
		msgMissingPort:          errors.New("port missing from request"),
		msgInvalidPort:          errors.New("Invalid port"),
		msgInvalidAuth:          errors.New("Invalid passkey"),
		msgInvalidInfoHash:      errors.New("Invalid info hash"),
		msgInvalidPeerID:        errors.New("Peer ID invalid"),
		msgInvalidNumWant:       errors.New("num_want invalid"),
		msgBadClient:            errors.New("Client not whitelisted"),
		msgInfoHashNotFound:     errors.New("Unknown infohash"),
		msgClientRequestTooFast: errors.New("Slow down there jimmy"),
		msgMalformedRequest:     errors.New("Malformed request"),
		msgGenericError:         errors.New("Generic error"),
		msgQueryParseFail:       errors.New("Could not parse request"),
	}
)

// Err maps a tracker error code to a error
//noinspection GoUnusedExportedFunction
//func Err(code errCode) error {
//	return responseStringMap[code]
//}

// getIP Parses and returns a IP from a query
// If a IP header exists, it will be used instead of the client provided query parameter
// If no query IP is provided, the
func getIP(q *query, allowClientIP bool, c *gin.Context) (net.IP, bool, error) {
	if allowClientIP {
		for i, k := range [3]announceParam{paramIP, paramIPv4, paramIPv6} {
			// Use client provided IP
			ipStr, found := q.Params[k]
			if found {
				switch i {
				case 0:
					return net.ParseIP(ipStr), false, nil
				case 1:
					return net.ParseIP(ipStr), false, nil
				case 2:
					return net.ParseIP(ipStr), true, nil
				}

			}
		}
	}
	// Look for forwarded ip in headers
	for _, header := range []string{"X-Real-IP", "X-Forwarded-For"} {
		if headerIP := c.Request.Header.Get(header); headerIP != "" {
			addr := net.ParseIP(headerIP)
			return addr, strings.Count(headerIP, ":") > 1, nil
		}
	}
	httpAddr, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	for i := 0; i < len(httpAddr); i++ {
		switch httpAddr[i] {
		case '.':
			return net.ParseIP(httpAddr), false, nil
		case ':':
			return net.ParseIP(httpAddr), true, nil
		}
	}
	return net.IP{}, false, consts.ErrInvalidClient
}

// oops will output a bencoded error code to the torrent client using
// a preset message code constant
func oops(ctx *gin.Context, errCode errCode) {
	msg, exists := responseStringMap[errCode]
	if !exists {
		msg = responseStringMap[msgGenericError]
	}
	ctx.Data(int(errCode), gin.MIMEPlain, responseError(msg.Error()))
	log.Errorf("Error in request from: %s (%d : %s)", ctx.Request.RequestURI, errCode, msg.Error())
}

// preFlightChecks ensures our user meets the requirements to make an authorized request
// THis is used within the request handler itself and not as a middleware because of the
// slightly higher cost of passing data in through the request context
func preFlightChecks(usr *store.User, pk string, c *gin.Context) bool {
	// Check that the user is valid before parsing anything
	if config.Tracker.Public {
		usr.UserID = 1
		return true
	} else {
		if pk == "" {
			oops(c, msgInvalidAuth)
			return false
		}
		foundUser, err := UserGetByPasskey(pk)
		if err != nil {
			log.Debugf("Got invalid passkey")
			oops(c, msgInvalidAuth)
			return false
		}
		usr = foundUser
		return usr.Valid()
	}
}

// handleTrackerErrors is used as the default error handler for tracker requests
// the error is returned to the client as a bencoded error string as defined in the
// bittorrent specs.
func handleTrackerErrors(ctx *gin.Context) {
	// Run request handler
	ctx.Next()
	// Handle any errors recorded
	errorReturned := ctx.Errors.Last()
	if errorReturned != nil {
		meta := errorReturned.JSON().(gin.H)
		status := msgGenericError
		customStatus, found := meta["status"]
		if found {
			status = customStatus.(errCode)
		}
		oops(ctx, status)
	}
}

// responseError generates a bencoded error response for the torrent client to
// parse and display to the user
//
// Note that this function does not generate or support a warning reason, which are rarely if
// ever used.
func responseError(message string) []byte {
	var buf bytes.Buffer
	if err := bencode.NewEncoder(&buf).Encode(bencode.Dict{
		"failure reason": message,
	}); err != nil {
		log.Errorf("Failed to encode error response: %s", err)
	}
	return buf.Bytes()
}

// newRouter creates and returns a newly configured router instance using
// the default middleware handlers.
func newRouter() *gin.Engine {
	router := gin.New()
	router.Use(ginlogrus.Logger(log.New()), gin.Recovery())
	return router
}

func noRoute(c *gin.Context) {
	c.String(http.StatusNotFound, "☃")
}

// NewBitTorrentHandler configures a router to handle tracker announce/scrape requests
func NewBitTorrentHandler() *gin.Engine {
	r := newRouter()
	r.Use(handleTrackerErrors)
	r.GET("/announce", announce)
	r.GET("/scrape", scrape)
	r.GET("/announce/:passkey", announce)
	r.GET("/scrape/:passkey", scrape)
	r.NoRoute(noRoute)
	return r
}

// HTTPOpts is used to configure a http.Server instance
type HTTPOpts struct {
	ListenAddr     string
	UseTLS         bool
	Handler        http.Handler
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	TLSConfig      *tls.Config
}

// DefaultHTTPOpts returns a default set of options for http.Server instances
func DefaultHTTPOpts() *HTTPOpts {
	return &HTTPOpts{
		ListenAddr:     ":34000",
		UseTLS:         false,
		Handler:        nil,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      nil,
	}
}

// NewHTTPServer will configure and return a *http.Server suitable for serving requests.
// This should be used over the default ListenAndServe options as they do not set certain
// parameters, notably timeouts, which can negatively effect performance.
func NewHTTPServer(opts *HTTPOpts) *http.Server {
	var tlsCfg *tls.Config
	if opts.UseTLS && opts.TLSConfig == nil {
		tlsCfg = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
	} else {
		tlsCfg = nil
	}
	srv := &http.Server{
		Addr:           opts.ListenAddr,
		Handler:        opts.Handler,
		TLSConfig:      tlsCfg,
		ReadTimeout:    opts.ReadTimeout,
		WriteTimeout:   opts.WriteTimeout,
		MaxHeaderBytes: opts.MaxHeaderBytes,
	}
	return srv
}
