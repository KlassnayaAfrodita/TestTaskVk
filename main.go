// main.go

package main

import (
	"fmt"
	"time"
	"workerpool"
)

func main() {
	manager := workerpool.NewManager(2, 10)

	manager.Start()

	// Добавляем задачи с задержкой
	for i := 0; i < 100; i++ {
		manager.AddTask(fmt.Sprintf("Task %d", i))
		time.Sleep(200 * time.Millisecond)
	}

	manager.Stop()
	fmt.Println("All tasks completed, exiting program.")
}
