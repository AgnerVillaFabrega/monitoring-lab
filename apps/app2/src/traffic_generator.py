#!/usr/bin/env python3

import json
import os
import random
import requests
import time
import threading
from datetime import datetime
from typing import List, Dict

class TrafficGenerator:
    def __init__(self):
        self.target_url = os.getenv("TARGET_URL", "http://app2-service:8000")
        self.request_interval = int(os.getenv("REQUEST_INTERVAL", "4"))
        self.error_rate = float(os.getenv("ERROR_RATE", "0.12"))
        
        self.endpoints = [
            {"path": "/health", "weight": 0.3, "method": "GET"},
            {"path": "/api/data", "weight": 0.4, "method": "GET"},
            {"path": "/api/compute", "weight": 0.2, "method": "GET"},
            {"path": "/api/database", "weight": 0.1, "method": "GET"}
        ]
        
        self.session = requests.Session()
        self.session.timeout = 10
    
    def log_message(self, level: str, message: str, **kwargs):
        log_entry = {
            "timestamp": datetime.utcnow().isoformat(),
            "level": level,
            "service": "app2-traffic-generator",
            "message": message,
            **kwargs
        }
        print(json.dumps(log_entry))
    
    def select_endpoint(self) -> Dict:
        """Selecciona un endpoint basado en los pesos configurados"""
        r = random.random()
        cumulative_weight = 0
        
        for endpoint in self.endpoints:
            cumulative_weight += endpoint["weight"]
            if r <= cumulative_weight:
                return endpoint
        
        return self.endpoints[0]  # fallback
    
    def make_request(self, endpoint: Dict):
        url = f"{self.target_url}{endpoint['path']}"
        
        try:
            start_time = time.time()
            response = self.session.request(endpoint["method"], url)
            duration = time.time() - start_time
            
            status = "success" if response.status_code < 400 else "error"
            
            self.log_message(
                "info",
                f"Request completed",
                endpoint=endpoint["path"],
                method=endpoint["method"],
                status_code=response.status_code,
                duration_ms=round(duration * 1000, 2),
                status=status
            )
            
        except requests.exceptions.RequestException as e:
            self.log_message(
                "error",
                f"Request failed: {str(e)}",
                endpoint=endpoint["path"],
                method=endpoint["method"],
                error_type=type(e).__name__
            )
    
    def generate_burst_traffic(self):
        """Genera ráfagas de tráfico para simular picos de carga"""
        while True:
            # Esperar entre 30-120 segundos para la próxima ráfaga
            wait_time = random.randint(30, 120)
            time.sleep(wait_time)
            
            # Generar ráfaga de 5-15 requests
            burst_size = random.randint(5, 15)
            
            self.log_message(
                "info",
                f"Generating traffic burst",
                burst_size=burst_size
            )
            
            threads = []
            for _ in range(burst_size):
                endpoint = self.select_endpoint()
                thread = threading.Thread(target=self.make_request, args=(endpoint,))
                threads.append(thread)
                thread.start()
                
                # Pequeña pausa entre requests de la ráfaga
                time.sleep(random.uniform(0.1, 0.5))
            
            # Esperar que terminen todos los threads
            for thread in threads:
                thread.join()
    
    def generate_regular_traffic(self):
        """Genera tráfico regular y constante"""
        self.log_message(
            "info",
            "Traffic generator started",
            target_url=self.target_url,
            request_interval=self.request_interval
        )
        
        while True:
            try:
                # Número de requests simultáneos (1-3)
                concurrent_requests = random.randint(1, 3)
                
                threads = []
                for _ in range(concurrent_requests):
                    endpoint = self.select_endpoint()
                    thread = threading.Thread(target=self.make_request, args=(endpoint,))
                    threads.append(thread)
                    thread.start()
                    
                    # Pausa muy pequeña entre iniciar threads
                    if concurrent_requests > 1:
                        time.sleep(random.uniform(0.1, 0.3))
                
                # Esperar que terminen
                for thread in threads:
                    thread.join()
                
                # Esperar hasta el próximo ciclo
                base_interval = self.request_interval
                jitter = random.uniform(-1, 1)  # ±1 segundo de jitter
                sleep_time = max(1, base_interval + jitter)
                time.sleep(sleep_time)
                
            except KeyboardInterrupt:
                self.log_message("info", "Traffic generator stopping")
                break
            except Exception as e:
                self.log_message(
                    "error",
                    f"Unexpected error in traffic generator: {str(e)}",
                    error_type=type(e).__name__
                )
                time.sleep(5)  # Breve pausa antes de continuar
    
    def run(self):
        """Ejecuta ambos generadores de tráfico en paralelo"""
        # Iniciar el generador de ráfagas en un thread separado
        burst_thread = threading.Thread(target=self.generate_burst_traffic, daemon=True)
        burst_thread.start()
        
        # Ejecutar el tráfico regular en el thread principal
        self.generate_regular_traffic()

if __name__ == "__main__":
    generator = TrafficGenerator()
    generator.run()