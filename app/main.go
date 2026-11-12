package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

func main() {
	appName := getEnv("APP_NAME", "OpsPulse")
	appVersion := getEnv("APP_VERSION", "1.0.0")
	environment := getEnv("APP_ENV", "development")
	port := getEnv("APP_PORT", "8080")
	deploymentTime := time.Now().Format(time.RFC1123)

	handler := newHandler(appName, appVersion, environment, deploymentTime)

	serverAddress := ":" + port
	log.Printf("%s starting on %s", appName, serverAddress)

	if err := http.ListenAndServe(serverAddress, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func newHandler(appName, appVersion, environment, deploymentTime string) http.Handler {
	mux := http.NewServeMux()

	var requestsTotal atomic.Uint64
	startTime := time.Now()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		requestsTotal.Add(1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%s | Deployment Dashboard</title>
	<style>
		body {
			margin: 0;
			font-family: Arial, sans-serif;
			background: #101827;
			color: #f1f5f9;
		}
		.container {
			max-width: 850px;
			margin: 70px auto;
			padding: 30px;
		}
		h1 { color: #38bdf8; }
		.subtitle { color: #94a3b8; }
		.card {
			background: #1e293b;
			padding: 24px;
			margin-top: 25px;
			border-radius: 12px;
		}
		.status { color: #4ade80; font-weight: bold; }
		.footer { color: #94a3b8; margin-top: 30px; }
	</style>
</head>
<body>
	<div class="container">
		<h1>%s</h1>
		<p class="subtitle">Go-powered deployment and monitoring platform</p>
		<div class="card">
			<h2>Application Status</h2>
			<p class="status">● Application Running</p>
			<p><strong>Version:</strong> %s</p>
			<p><strong>Environment:</strong> %s</p>
			<p><strong>Deployment Time:</strong> %s</p>
		</div>
		<p class="footer">Powered by Go | GoDeploy Project</p>
	</div>
</body>
</html>`, appName, appName, appVersion, environment, deploymentTime)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "healthy")
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		fmt.Fprintf(w, "# HELP godeploy_app_info Application information\n")
		fmt.Fprintf(w, "# TYPE godeploy_app_info gauge\n")
		fmt.Fprintf(w, "godeploy_app_info{app_name=\"%s\",version=\"%s\",environment=\"%s\"} 1\n",
			appName, appVersion, environment)

		fmt.Fprintf(w, "# HELP godeploy_http_requests_total Total homepage requests\n")
		fmt.Fprintf(w, "# TYPE godeploy_http_requests_total counter\n")
		fmt.Fprintf(w, "godeploy_http_requests_total %d\n", requestsTotal.Load())

		fmt.Fprintf(w, "# HELP godeploy_app_uptime_seconds Application uptime in seconds\n")
		fmt.Fprintf(w, "# TYPE godeploy_app_uptime_seconds gauge\n")
		fmt.Fprintf(w, "godeploy_app_uptime_seconds %.2f\n", time.Since(startTime).Seconds())
	})

	return mux
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
