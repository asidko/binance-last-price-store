package http

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"binance-tick-store/internal/database"
)

// StatusProvider provides current connection status.
type StatusProvider interface {
	GetActiveSymbols() map[string]bool
}

// Handler handles HTTP requests.
type Handler struct {
	store     database.Store
	status    StatusProvider
	startTime time.Time
}

// NewHandler creates a new HTTP handler.
func NewHandler(store database.Store, status StatusProvider) *Handler {
	return &Handler{
		store:     store,
		status:    status,
		startTime: time.Now(),
	}
}

// ServeHTTP handles the /status endpoint.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/status" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	uptime := formatDuration(time.Since(h.startTime))
	active := h.status.GetActiveSymbols()

	var sb strings.Builder
	sb.WriteString("Binance Last Price Store\n\n")
	sb.WriteString(fmt.Sprintf("Status:     running\n"))
	sb.WriteString(fmt.Sprintf("Uptime:     %s\n\n", uptime))

	settings, err := h.store.GetSymbolSettings()
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		w.Write([]byte(sb.String()))
		return
	}

	if len(settings) == 0 {
		sb.WriteString("No symbols configured\n")
		w.Write([]byte(sb.String()))
		return
	}

	// Sort symbols alphabetically
	sort.Slice(settings, func(i, j int) bool {
		return settings[i].Symbol < settings[j].Symbol
	})

	for _, s := range settings {
		status := "off"
		if s.Enabled && active[s.Symbol] {
			status = "on"
		}

		dateRange := "(no data yet)"
		count, _ := h.store.GetCount(s.Symbol)
		dr, err := h.store.GetDateRange(s.Symbol)
		if err == nil && dr.From != nil && dr.To != nil {
			dateRange = fmt.Sprintf("%s  ->  %s",
				dr.From.Format("2006-01-02 15:04:05"),
				dr.To.Format("2006-01-02 15:04:05"))
		}

		sb.WriteString(fmt.Sprintf("%-12s%-8s%-10d%s\n", s.Symbol, status, count, dateRange))
	}

	w.Write([]byte(sb.String()))
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
