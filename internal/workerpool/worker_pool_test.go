// worker_pool_test.go

package workerpool

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// Тест минимального количества воркеров
func TestMinWorkers(t *testing.T) {
	workerPool := NewWorkerPool(2, 5)
	workerPool.Start()

	if atomic.LoadInt32(&workerPool.workerCount) != 2 {
		t.Errorf("Expected 2 workers at start, got %d", workerPool.workerCount)
	}

	workerPool.Stop()
}

// Тест максимального количества воркеров при высокой нагрузке
func TestMaxWorkers(t *testing.T) {
	workerPool := NewWorkerPool(2, 5)
	workerPool.Start()

	for i := 0; i < 100; i++ {
		workerPool.AddTask(fmt.Sprintf("Task %d", i))
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-time.After(3 * time.Second):
			t.Errorf("Expected 5 workers, but got %d", atomic.LoadInt32(&workerPool.workerCount))
			return
		case <-ticker.C:
			if atomic.LoadInt32(&workerPool.workerCount) == 5 {
				fmt.Println("Reached maximum number of workers: 5")
				workerPool.Stop()
				return
			}
		}
	}
}

// Тест на завершение всех задач
func TestAllTasksCompleted(t *testing.T) {
	workerPool := NewWorkerPool(2, 5)
	workerPool.Start()

	for i := 0; i < 5; i++ {
		workerPool.AddTask(fmt.Sprintf("Task %d", i))
	}

	time.Sleep(3 * time.Second)

	if len(workerPool.taskQueue) != 0 {
		t.Errorf("Expected all tasks to be completed, but %d tasks remain", len(workerPool.taskQueue))
	}

	workerPool.Stop()
}

// Тест автоматического уменьшения числа воркеров
func TestAutoScaleDown(t *testing.T) {
	workerPool := NewWorkerPool(2, 5)
	workerPool.Start()

	for i := 0; i < 10; i++ {
		workerPool.AddTask(fmt.Sprintf("Task %d", i))
		time.Sleep(100 * time.Millisecond)
	}

	// ждем, пока задачи поступят и начнут исполнятся
	time.Sleep(2 * time.Second)

	initialWorkers := atomic.LoadInt32(&workerPool.workerCount)
	if initialWorkers < 2 || initialWorkers > 5 {
		t.Errorf("Unexpected worker count after scaling up: got %d, want between %d and %d", initialWorkers, 2, 5)
	}

	time.Sleep(3 * time.Second)

	// ждем, пока выполнятся задачи и воркеры удалятся
	finalWorkers := atomic.LoadInt32(&workerPool.workerCount)
	if finalWorkers != 2 {
		t.Errorf("Expected worker count to scale down to 2, but got %d", finalWorkers)
	}

	workerPool.Stop()
	fmt.Println("TestAutoScaleDown completed")
}

// Тест на остановку воркер пула
func TestStopAllWorkers(t *testing.T) {
	workerPool := NewWorkerPool(2, 5)
	workerPool.Start()

	for i := 0; i < 5; i++ {
		workerPool.AddTask(fmt.Sprintf("Task %d", i))
	}

	workerPool.Stop()

	// Проверяем, что все воркеры остановлены
	activeWorkers := atomic.LoadInt32(&workerPool.workerCount)
	if activeWorkers != 0 {
		t.Errorf("Expected all workers to be stopped, but %d workers are still active", activeWorkers)
	} else {
		fmt.Println("All workers stopped as expected")
	}
}
