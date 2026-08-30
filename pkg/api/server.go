package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/projectdiscovery/gologger"
	"v2ray_checker/pkg/model"
	"v2ray_checker/pkg/storage"
)

type Server struct {
	store *storage.Store
	port  string
}

func NewServer(port string, store *storage.Store) *Server {
	return &Server{
		store: store,
		port:  port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// 1. JSON API endpoint requested by user
	mux.HandleFunc("/api/nodes", s.handleGetNodesJSON)
	mux.HandleFunc("/nodes.json", s.handleGetNodesJSON)

	// 2. Base64 Raw Subscription endpoints for client apps
	mux.HandleFunc("/sub/all", s.handleSubAll)
	mux.HandleFunc("/sub", s.handleSubAll)
	mux.HandleFunc("/sub/vless", s.handleSubProto("vless"))
	mux.HandleFunc("/sub/vmess", s.handleSubProto("vmess"))
	mux.HandleFunc("/sub/trojan", s.handleSubProto("trojan"))
	mux.HandleFunc("/sub/ss", s.handleSubProto("ss"))

	// 3. Stats and Health
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/stats", s.handleStats)

	gologger.Info().Msgf("HTTP REST API and Subscription server listening on %s", s.port)
	return http.ListenAndServe(s.port, mux)
}

// GET /api/nodes?proto=vless&country=IR
// Returns exact user format: [{"config":"...","ping":"62","country":"IR","ip":"185.243.45.15"}]
func (s *Server) handleGetNodesJSON(w http.ResponseWriter, r *http.Request) {
	proto := r.URL.Query().Get("proto")
	nodes, err := s.store.GetActiveNodes(proto)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	countryFilter := strings.ToUpper(r.URL.Query().Get("country"))
	var response []model.UserNodeResponse

	for _, n := range nodes {
		if countryFilter != "" && strings.ToUpper(n.Country) != countryFilter {
			continue
		}
		response = append(response, model.UserNodeResponse{
			Config:  n.Config,
			Ping:    n.Ping,
			Country: n.Country,
			IP:      n.IP,
		})
	}

	if response == nil {
		response = []model.UserNodeResponse{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(response)
}

// GET /sub/all or /sub
// Returns pure Base64 encoded list of only active and tested configs
func (s *Server) handleSubAll(w http.ResponseWriter, r *http.Request) {
	s.serveSubscription(w, "")
}

func (s *Server) handleSubProto(proto string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.serveSubscription(w, proto)
	}
}

func (s *Server) serveSubscription(w http.ResponseWriter, proto string) {
	nodes, err := s.store.GetActiveNodes(proto)
	if err != nil {
		http.Error(w, "Failed to retrieve active nodes", http.StatusInternalServerError)
		return
	}

	var rawConfigs []string
	for _, n := range nodes {
		// Include country tag in node remarks if not present
		cleanConfig := n.Config
		if n.Country != "" && !strings.Contains(cleanConfig, "#") {
			cleanConfig = fmt.Sprintf("%s#%s_%sms", cleanConfig, n.Country, n.Ping)
		}
		rawConfigs = append(rawConfigs, cleanConfig)
	}

	payload := strings.Join(rawConfigs, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Subscription-Userinfo", fmt.Sprintf("upload=0; download=0; total=107374182400; expire=%d", 2000000000))
	w.Write([]byte(encoded))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","service":"v2ray_checker"}`))
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
