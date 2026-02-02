package http_server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/shiwa/timecard-mini/extracted-source/config"
	"github.com/shiwa/timecard-mini/extracted-source/interactive/api"
)

// HttpServer по дизассемблеру daemon: сервер HTTP для interactive API (outputTimeSourcesStatus 0x4c26d60, outputFormattedJSON 0x4c27720).
// Writer — опциональный вывод для JSON (например http.ResponseWriter); при nil OutputFormattedJSON только сериализует.
type HttpServer struct {
	Writer io.Writer
}

// OutputTimeSourcesStatus по дизассемблеру (0x4c26d60): GetHTTPTimeSourcesStatus(); convTslice; outputFormattedJSON(type, data). Данные из api (hostclocks + sources).
func (h *HttpServer) OutputTimeSourcesStatus() {
	data := api.GetHTTPTimeSourcesStatus()
	h.OutputFormattedJSON(nil, data)
}

// OutputFormattedJSON по дизассемблеру (0x4c27720): сериализация typ и data в JSON и отправка в Writer (если задан).
func (h *HttpServer) OutputFormattedJSON(typ interface{}, data interface{}) {
	payload := struct {
		Type interface{} `json:"type,omitempty"`
		Data interface{} `json:"data"`
	}{Type: typ, Data: data}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if h != nil && h.Writer != nil {
		_, _ = h.Writer.Write(raw)
	}
}

// Run по дизассемблеру (0x4c26a00): appConfig+0x370/0x378 host/port; net.Listen; http.Serve с handlers для /api/time-sources и др.
func (h *HttpServer) Run() {
	addr := ":8080"
	if cfg := config.GetAppConfig(); cfg != nil {
		host := cfg.HTTPListenAddr
		if host == "" {
			host = "0.0.0.0"
		}
		port := cfg.HTTPPort
		if port == 0 {
			port = 8080
		}
		addr = fmt.Sprintf("%s:%d", host, port)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/time-sources", h.handleTimeSources)
	mux.HandleFunc("/", h.handleRoot)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	defer listener.Close()
	_ = http.Serve(listener, mux)
}

// handleRoot — корневой handler (заглушка).
func (h *HttpServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("timebeat\n"))
}

// handleTimeSources — HTTP handler: OutputTimeSourcesStatus в ResponseWriter.
func (h *HttpServer) handleTimeSources(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	h2 := &HttpServer{Writer: w}
	h2.OutputTimeSourcesStatus()
}

func Extracted_Go() {}

