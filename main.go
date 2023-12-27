package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func worker() {
	for {
		fmt.Printf("Hello from worker at %v\n", time.Now())
		time.Sleep(20 * time.Second)
	}

}

func watchdog(process *os.Process, restartChan chan struct{}, heartbeat chan time.Time) {
	timeout := 30 * time.Second
	timer := time.NewTimer(timeout)

	for {
		select {
		case beatTime := <-heartbeat:
			fmt.Printf("Heartbeat received at %v\n", beatTime) // do nothing
			timer.Reset(timeout)
		case <-timer.C:
			fmt.Println("Process is not alive, restarting")
			process.Kill()
			filepath := "dead.txt"
			_, err := os.Stat(filepath)
			if os.IsNotExist(err) {
				break
			}

			// Spawn a new instance of the worker process
			newProcess := exec.Command(os.Args[0], "worker")
			newProcess.Stdout = os.Stdout
			newProcess.Stderr = os.Stderr
			newProcess.Start()

			// Notify the watchdog about the new process
			restartChan <- struct{}{}
			process = newProcess.Process

		}

	}
}

func main() {

	if len(os.Args) > 1 && os.Args[1] == "worker" {
		go worker()
	}

	restartChan := make(chan struct{})
	defer close(restartChan)

	// start the initial worker process
	cmd := exec.Command(os.Args[0], "worker")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()

	heartbeat := make(chan time.Time)
	defer close(heartbeat)
	go func() {
		for i := 0; i < 3; i++ {
			heartbeat <- time.Now()
			time.Sleep(5 * time.Second)
		}
	}()

	go watchdog(cmd.Process, restartChan, heartbeat)
	<-restartChan
	cmd.Wait()
}
