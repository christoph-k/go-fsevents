package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/christoph-k/go-fsevents"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %v <dir|file>\n\n", os.Args[0])
		os.Exit(1)
	}
	_, err := os.Stat(os.Args[1])
	if err != nil {
		fmt.Printf("%v\nusage: %v <dir|file>\n\n", err, os.Args[0])
		os.Exit(1)
	}
	w, err := fsevents.NewWatcher(os.Args[1], 200*time.Millisecond)
	if err != nil {
		panic(err)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigChan
		w.Stop()
		os.Exit(0)
	}()

	for {
		e := <-w.EventChan
		switch e.EventType {
		case fsevents.FSEventCreate:
			fmt.Printf("%v created\n", e.Path)
		case fsevents.FSEventDelete:
			fmt.Printf("%v deleted\n", e.Path)
		case fsevents.FSEventModify:
			fmt.Printf("%v modified\n", e.Path)
		}
	}
}
