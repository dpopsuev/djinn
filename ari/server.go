package ari

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// Server routes.
const (
	RouteIntent      = "/intent"
	RouteAndon       = "/andon"
	RouteWorkstreams = "/workstreams"
	RouteCordonClear = "/cordon/clear"
	RouteEvents      = "/events"
)

// SSE event type names.
const (
	eventTypeProgress   = "progress"
	eventTypePermission = "permission"
	eventTypeAndon      = "andon"
	eventTypeResult     = "result"
)

// HTTP response status values.
const (
	responseAccepted = "accepted"
	responseCleared  = "cleared"
)

// HTTP error messages.
const (
	errMethodNotAllowed = "method not allowed"
	errInvalidBody      = "invalid request body"
	errStreamingUnsupported = "streaming not supported"
)

// HTTP headers.
const (
	headerContentType   = "Content-Type"
	headerCacheControl  = "Cache-Control"
	headerConnection    = "Connection"
	contentTypeJSON     = "application/json"
	contentTypeSSE      = "text/event-stream"
	cacheControlNoCache = "no-cache"
	connectionKeepAlive = "keep-alive"
)

// Server is an HTTP server that bridges an ARI Runtime to operators.
// It implements OperatorPort: operators send intents and receive events
// via HTTP + Server-Sent Events.
type Server struct {
	runtime  Runtime
	mux      *http.ServeMux
	server   *http.Server

	mu          sync.Mutex
	intentHandler func(Intent)
	eventCh     chan serverEvent
	permRespCh  chan PermissionResponse
}

type serverEvent struct {
	Type string      `json:"type"`
	Data any `json:"data"`
}

// NewServer creates a new ARI HTTP server.
func NewServer(rt Runtime) *Server {
	mux := http.NewServeMux()
	s := &Server{
		runtime:    rt,
		mux:        mux,
		server:     &http.Server{Handler: mux},
		eventCh:    make(chan serverEvent, 100),
		permRespCh: make(chan PermissionResponse, 10),
	}
	mux.HandleFunc(RouteIntent, s.handleIntent)
	mux.HandleFunc(RouteAndon, s.handleAndon)
	mux.HandleFunc(RouteWorkstreams, s.handleWorkstreams)
	mux.HandleFunc(RouteCordonClear, s.handleCordonClear)
	mux.HandleFunc(RouteEvents, s.handleEvents)
	return s
}

// Start begins listening on the given address. Blocks until stopped.
func (s *Server) Start(addr string) error {
	s.server.Addr = addr
	return s.server.ListenAndServe()
}

// StartOnListener begins serving on the given listener. Blocks until stopped.
func (s *Server) StartOnListener(ln net.Listener) error {
	return s.server.Serve(ln)
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// --- OperatorPort implementation ---

// OnIntent registers the handler called when an intent arrives via HTTP.
func (s *Server) OnIntent(handler func(Intent)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intentHandler = handler
}

// EmitProgress sends a progress event to connected SSE clients.
func (s *Server) EmitProgress(event any) {
	s.emit(eventTypeProgress, event)
}

// EmitPermission sends a permission event to connected SSE clients.
func (s *Server) EmitPermission(payload PermissionPayload) {
	s.emit(eventTypePermission, payload)
}

// EmitAndon sends an andon event to connected SSE clients.
func (s *Server) EmitAndon(board any) {
	s.emit(eventTypeAndon, board)
}

// EmitResult sends a result event to connected SSE clients.
func (s *Server) EmitResult(result Result) {
	s.emit(eventTypeResult, result)
}

// PermissionResponses returns the channel for permission responses from operators.
func (s *Server) PermissionResponses() <-chan PermissionResponse {
	return s.permRespCh
}

func (s *Server) emit(eventType string, data any) {
	select {
	case s.eventCh <- serverEvent{Type: eventType, Data: data}:
	default:
		// Drop if buffer full — SSE clients will miss events
	}
}

// --- HTTP handlers ---

func (s *Server) handleIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var intent Intent
	if err := json.NewDecoder(r.Body).Decode(&intent); err != nil {
		http.Error(w, errInvalidBody, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	h := s.intentHandler
	s.mu.Unlock()

	if h != nil {
		h(intent)
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(map[string]string{"status": responseAccepted})
}

func (s *Server) handleAndon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	board := s.runtime.Andon()
	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(board)
}

func (s *Server) handleWorkstreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ws := s.runtime.ListWorkstreams()
	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(ws)
}

func (s *Server) handleCordonClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, errInvalidBody, http.StatusBadRequest)
		return
	}
	s.runtime.ClearCordon(req.Paths)
	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(map[string]string{"status": responseCleared})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, errStreamingUnsupported, http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, contentTypeSSE)
	w.Header().Set(headerCacheControl, cacheControlNoCache)
	w.Header().Set(headerConnection, connectionKeepAlive)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-s.eventCh:
			if !ok {
				return
			}
			data, _ := json.Marshal(evt)
			w.Write([]byte("data: "))
			w.Write(data)
			w.Write([]byte("\n\n"))
			flusher.Flush()
		case <-time.After(30 * time.Second):
			// Keepalive
			w.Write([]byte(": keepalive\n\n"))
			flusher.Flush()
		}
	}
}
