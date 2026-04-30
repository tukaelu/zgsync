package zendesk

import (
	"net/http"
	"time"
)

// initializeDefaultScenarios sets up predefined behavior scenarios
func (sm *ScenarioManager) initializeDefaultScenarios() {
	sm.behaviors["AuthFailure"] = &BehaviorSet{
		Name:       "AuthFailure",
		ErrorRate:  0.0,
		LatencyMin: 5 * time.Millisecond,
		LatencyMax: 20 * time.Millisecond,
		FailurePatterns: map[string]ErrorPattern{
			"auth_failure": {
				StatusCode: http.StatusUnauthorized,
				Response:   `{"error": "Unauthorized", "description": "Authentication credentials invalid"}`,
				Condition: func(r *http.Request) bool {
					// Always fail authentication in this scenario
					return true
				},
			},
		},
		CustomHandlers: make(map[string]http.HandlerFunc),
	}
}
