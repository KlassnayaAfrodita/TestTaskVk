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
	manager := NewManager(2, 5)
	manager.Start()

	if atomic.LoadInt32(&manager.workerCount) != 2 {
		t.Errorf("Expected 2 workers at start, got %d", manager.workerCount)
	}

	manager.Stop()
}

// Тест максимального количества воркеров при высокой нагрузке
func TestMaxWorkers(t *testing.T) {
	manager := NewManager(2, 5)
	manager.Start()

	// Добавляем задачи в пул, чтобы вызвать автоматическое увеличение количества воркеров
	for i := 0; i < 100; i++ {
		manager.AddTask(fmt.Sprintf("Task %d", i))
	}

	// Ожидаем, пока количество воркеров не достигнет максимума (5), с таймаутом на случай, если это не происходит
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-time.After(3 * time.Second):
			t.Errorf("Expected 5 workers, but got %d", atomic.LoadInt32(&manager.workerCount))
			return
		case <-ticker.C:
			if atomic.LoadInt32(&manager.workerCount) == 5 {
				fmt.Println("Reached maximum number of workers: 5")
				manager.Stop()
				return
			}
		}
	}
}

// Тест на завершение всех задач
func TestAllTasksCompleted(t *testing.T) {
	manager := NewManager(2, 5)
	manager.Start()

	for i := 0; i < 5; i++ {
		manager.AddTask(fmt.Sprintf("Task %d", i))
	}

	// Даем время на завершение задач
	time.Sleep(3 * time.Second)

	if len(manager.taskQueue) != 0 {
		t.Errorf("Expected all tasks to be completed, but %d tasks remain", len(manager.taskQueue))
	}

	manager.Stop()
}

// Тест автоматического уменьшения числа воркеров
func TestAutoScaleDown(t *testing.T) {
	manager := NewManager(2, 5) // Настраиваем менеджер с 2-5 воркерами
	manager.Start()

	// Добавляем несколько задач, чтобы вызвать авто-масштабирование
	for i := 0; i < 10; i++ {
		manager.AddTask(fmt.Sprintf("Task %d", i))
		time.Sleep(100 * time.Millisecond)
	}

	// Ждем некоторое время, чтобы позволить воркерам начать обработку задач
	time.Sleep(2 * time.Second)

	// Проверяем количество воркеров
	initialWorkers := atomic.LoadInt32(&manager.workerCount)
	if initialWorkers < 2 || initialWorkers > 5 {
		t.Errorf("Unexpected worker count after scaling up: got %d, want between %d and %d", initialWorkers, 2, 5)
	}

	// Ждем завершения задач, чтобы проверить авто-скейлдаун
	time.Sleep(3 * time.Second)

	// Убедимся, что воркеры уменьшились до минимального количества
	finalWorkers := atomic.LoadInt32(&manager.workerCount)
	if finalWorkers != 2 {
		t.Errorf("Expected worker count to scale down to 2, but got %d", finalWorkers)
	}

	// Остановка менеджера
	manager.Stop()
	fmt.Println("TestAutoScaleDown completed")
}

func TestStopAllWorkers(t *testing.T) {
	manager := NewManager(2, 5)
	manager.Start()

	// Добавляем задачи в пул
	for i := 0; i < 5; i++ {
		manager.AddTask(fmt.Sprintf("Task %d", i))
	}

	manager.Stop() // Останавливаем пул воркеров и дожидаемся завершения всех воркеров

	// Проверяем, что все воркеры остановлены
	activeWorkers := atomic.LoadInt32(&manager.workerCount)
	if activeWorkers != 0 {
		t.Errorf("Expected all workers to be stopped, but %d workers are still active", activeWorkers)
	} else {
		fmt.Println("All workers stopped as expected")
	}
}
