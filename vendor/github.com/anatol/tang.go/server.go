package tang

import (
	"io"
	"net/http"
	"strings"
)

// Server is a HTTP server instance that handles Tang exchange requests
type Server struct {
	http.Server
	Keys *KeySet
}

func (srv *Server) advertiseKey(w http.ResponseWriter, req *http.Request) {
	uri := req.RequestURI

	var thumbprint string
	if strings.HasPrefix(uri, "/adv/") {
		thumbprint = uri[5:]
	}

	if thumbprint != "" {
		key, found := srv.Keys.byThumbprint[thumbprint]
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if key.advertisement == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_, _ = w.Write(key.advertisement)
	} else {
		_, _ = w.Write(srv.Keys.DefaultAdvertisement)
	}
}

func (srv *Server) recoverKey(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	in, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	thp := req.RequestURI[5:]
	out, err := srv.Keys.Recover(thp, in)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/jwk+json")
	_, _ = w.Write(out)
}

func (srv *Server) handleRequest(w http.ResponseWriter, req *http.Request) {
	uri := req.RequestURI
	if uri == "/adv" || strings.HasPrefix(uri, "/adv/") {
		srv.advertiseKey(w, req)
	} else if strings.HasPrefix(uri, "/rec/") {
		srv.recoverKey(w, req)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// NewServer creates a new instance of http server that handles tang requests
func NewServer() *Server {
	var srv Server
	srv.Handler = http.HandlerFunc(srv.handleRequest)
	return &srv
}
