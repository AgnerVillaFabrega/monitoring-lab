package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type TrafficConfig struct {
	TargetURL       string `json:"target_url"`
	RequestInterval int    `json:"request_interval_seconds"`
	ErrorRate       float32 `json:"error_rate"`
}

func loadConfig() TrafficConfig {
	config := TrafficConfig{
		TargetURL:       "http://app1-service:8080",
		RequestInterval: 5,
		ErrorRate:       0.1,
	}
	
	if url := os.Getenv("TARGET_URL"); url != "" {
		config.TargetURL = url
	}
	
	return config
}

func makeRequest(url string, endpoint string) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(url + endpoint)
	if err != nil {
		log.Printf("Error making request to %s%s: %v", url, endpoint, err)
		return
	}
	defer resp.Body.Close()
	
	status := "success"
	if resp.StatusCode >= 400 {
		status = "error"
	}
	
	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     "info",
		"service":   "app1-traffic-generator",
		"message":   fmt.Sprintf("Request to %s - Status: %d", endpoint, resp.StatusCode),
		"endpoint":  endpoint,
		"status":    status,
	}
	
	logJSON, _ := json.Marshal(logEntry)
	fmt.Println(string(logJSON))
}

func generateTraffic() {
	config := loadConfig()
	
	endpoints := []string{"/health", "/data", "/slow"}
	weights := []float32{0.5, 0.4, 0.1} // Probabilidades relativas
	
	logEntry := map[string]interface{}{
		"timestamp":  time.Now().Format(time.RFC3339),
		"level":      "info",
		"service":    "app1-traffic-generator",
		"message":    "Traffic generator started",
		"target_url": config.TargetURL,
	}
	
	logJSON, _ := json.Marshal(logEntry)
	fmt.Println(string(logJSON))
	
	ticker := time.NewTicker(time.Duration(config.RequestInterval) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Seleccionar endpoint basado en pesos
			r := rand.Float32()
			var endpoint string
			
			if r < weights[0] {
				endpoint = endpoints[0]
			} else if r < weights[0]+weights[1] {
				endpoint = endpoints[1]
			} else {
				endpoint = endpoints[2]
			}
			
			// Generar múltiples requests para simular carga
			numRequests := 1 + rand.Intn(3) // 1-3 requests
			
			for i := 0; i < numRequests; i++ {
				go makeRequest(config.TargetURL, endpoint)
				
				// Pequeña pausa entre requests
				if i < numRequests-1 {
					time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)
				}
			}
		}
	}
}

func main() {
	// Seed para randomización
	rand.Seed(time.Now().UnixNano())
	
	generateTraffic()
}