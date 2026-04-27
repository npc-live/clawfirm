package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func evalServerPort() int {
	if p := os.Getenv("CLAWFIRM_EVAL_PORT"); p != "" {
		if port, err := strconv.Atoi(p); err == nil {
			return port
		}
	}
	return 9310
}

func (a *App) startEvalServer() {
	port := evalServerPort()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/eval", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Script string `json:"script"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		result := a.EvalJS(req.Script)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": result})
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("app: eval server: %v", err)
		}
	}()
	log.Printf("app: eval server listening on http://localhost:%d", port)
}
