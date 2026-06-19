package handlers

import (
	"net/http"
	"time"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/scoring"
)

// dashboardView is the data shape passed into web/templates/dashboard.html.
type dashboardView struct {
	Summary       scoring.Summary
	Results       []scoring.Result
	WeeklyEvents  []scoring.ScoreHistoryPoint
	BackendHost   string
	GeneratedAt   string
}

func (a *API) handleDashboard(w http.ResponseWriter, r *http.Request) {
	view := dashboardView{
		Summary:      a.svc.Summarize(),
		Results:      a.svc.ListScored(),
		WeeklyEvents: scoring.WeeklyMFAChallenges(time.Now()),
		BackendHost:  hostname(),
		GeneratedAt:  time.Now().Format("Jan 02, 2006 15:04 MST"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "dashboard.html", view); err != nil {
		http.Error(w, "template render error: "+err.Error(), http.StatusInternalServerError)
	}
}
