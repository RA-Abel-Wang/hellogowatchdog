package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func worker() {
	heartbeat := make(chan time.Time)
	defer close(heartbeat)

	restartChan := make(chan struct{})
	defer close(restartChan)

	go watchdog(restartChan, heartbeat)

	heartbeat_freq := 12 * time.Second
	timer := time.NewTimer(heartbeat_freq)

	for {
		select {
		case <-timer.C:
			fmt.Printf("10 seconds elapsed, sending heartbeat at %v\n", time.Now())
			heartbeat <- time.Now()
			timer.Reset(heartbeat_freq)
		case <-restartChan:
			fmt.Printf("Worker is restarting at %v\n", time.Now())
			return
		default:
			pid := os.Getpid()
			fmt.Printf("Doing some work at %v with pid %d\n", time.Now(), pid)
			time.Sleep(1 * time.Second)
		}

	}

}

func watchdog(restartChan chan struct{}, heartbeat chan time.Time) {
	timeout := 10 * time.Second
	timer := time.NewTimer(timeout)

	for {
		select {
		case beatTime := <-heartbeat:
			fmt.Printf("Heartbeat received at %v\n", beatTime) // do nothing
			timer.Reset(timeout)
		case <-timer.C:
			fmt.Println("Process is not alive, restarting")
			filepath := "dead.txt"
			_, err := os.Stat(filepath)
			if os.IsNotExist(err) {
				restartChan <- struct{}{}
				return
			}

			// Spawn a new instance of the worker process
			newProcess := exec.Command(os.Args[0], "worker")
			newProcess.Stdout = os.Stdout
			newProcess.Stderr = os.Stderr
			newProcess.Start()
			// Notify the watchdog about the new process
			restartChan <- struct{}{}
		}

	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		worker()
	}
	// start the initial worker process
	cmd := exec.Command(os.Args[0], "worker")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()

}
