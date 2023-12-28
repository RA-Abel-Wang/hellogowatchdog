package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func worker() {

	http.HandleFunc("/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
		pid := os.Getpid()
		fmt.Printf("Worker is handle heartbeat at %v with pid %d\n", time.Now(), pid)

	})

	http.HandleFunc("/greeting", func(w http.ResponseWriter, r *http.Request) {
		// Handle exit requests
		pid := os.Getpid()
		fmt.Printf("Worker is handle greeting at %v with pid %d\n", time.Now(), pid)
		// response
		w.Write([]byte("OK"))

	})

	http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
		// Handle exit requests
		pid := os.Getpid()
		fmt.Printf("Worker is handle exit at %v with pid %d\n", time.Now(), pid)
		// response
		w.Write([]byte("OK"))
	})

	// Start the HTTP server on port 8080
	err := http.ListenAndServe(":11111", nil)
	pid := os.Getpid()

	if err != nil {
		fmt.Printf("Error starting heartbeat server:at %v with pid %d\n", err, pid)
		return
	}

	fmt.Printf("starting heartbeat server:at %v with pid %d\n", err, pid)
}

func watchdog(process *os.Process, heartbeat chan time.Time) {
	timeout := 10 * time.Second
	timer := time.NewTimer(timeout)

	for {
		select {
		case beatTime := <-heartbeat:
			fmt.Printf("Heartbeat received at %v with pid %d\n", beatTime, process.Pid) // do nothing
			timer.Reset(timeout)
		case <-timer.C:
			fmt.Println("Process is not alive, restarting")
			process.Kill()
			// Spawn a new instance of the worker process
			newProcess := exec.Command(os.Args[0], "worker")
			newProcess.Stdout = os.Stdout
			newProcess.Stderr = os.Stderr
			err := newProcess.Start()
			if err != nil {
				fmt.Printf("Error starting process at: %v\n", err)
			} else {
				fmt.Printf("Process restarted with pid %d\n", newProcess.Process.Pid)
				process = newProcess.Process
			}
			timer.Reset(timeout)
		}

	}
}

func check_liveness(heartbeat chan time.Time) {
	// total 10 seconds
	heartbeat_freq := 5 * time.Second
	timer := time.NewTimer(heartbeat_freq)
	pid := os.Getpid()

	for {
		select {
		case <-timer.C:
			fmt.Printf("10 seconds elapsed, sending heartbeat at %v with the pid %d\n", time.Now(), pid)
			timeout := 5 * time.Second

			// Create an HTTP client with a timeout
			client := &http.Client{
				Timeout: timeout,
			}
			resp, err := client.Get("http://localhost:11111/heartbeat")
			if err != nil {
				fmt.Printf("Error in sending heartbeat at %v with the pid %d\n", err, pid)
			} else {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("Error in reading response body at %v with the pid %d\n", err, pid)
				} else {
					fmt.Printf("Response %s from the server at %v with the pid %d\n", string(body), time.Now(), pid)
					heartbeat <- time.Now()
				}

				defer resp.Body.Close()
			}
			timer.Reset(heartbeat_freq)
		}

	}

}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		worker()
	} else {
		// start the initial worker process
		cmd := exec.Command(os.Args[0], "worker")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()

		heartbeat := make(chan time.Time)
		defer close(heartbeat)
		go check_liveness(heartbeat)
		go watchdog(cmd.Process, heartbeat)

	}

	select {}

}
