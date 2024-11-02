package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Manager struct {
	taskQueue    chan string
	stopChan     chan struct{}
	workerCount  int32
	maxWorkers   int
	minWorkers   int
	wg           sync.WaitGroup
	mu           sync.Mutex
}

func NewManager(minWorkers, maxWorkers int) *Manager {
	manager := &Manager{
		taskQueue:  make(chan string, 100),
		stopChan:   make(chan struct{}), // Канал для остановки
		minWorkers: minWorkers,
		maxWorkers: maxWorkers,
	}

	// Запускаем минимальное количество воркеров
	for i := 0; i < minWorkers; i++ {
		manager.addWorker()
	}

	go manager.autoScaling() // Запускаем мониторинг нагрузки
	return manager
}

// Добавляет воркера в пул
func (m *Manager) addWorker() {
	workerID := atomic.AddInt32(&m.workerCount, 1)
	m.wg.Add(1)

	go func(id int32) {
		defer func() {
			atomic.AddInt32(&m.workerCount, -1)
			m.wg.Done()
		}()

		for {
			select {
			case task, ok := <-m.taskQueue:
				if !ok {
					return // Если канал задач закрыт, завершаем воркера
				}
				fmt.Printf("Worker %d processing task: %s\n", id, task)
				time.Sleep(500 * time.Millisecond) // Имитация обработки задачи
			case <-m.stopChan:
				return // Получен сигнал остановки, завершаем работу воркера
			}
		}
	}(workerID)
}

// Функция автоматического масштабирования воркеров
func (m *Manager) autoScaling() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			queueLen := len(m.taskQueue)
			currentWorkers := int(atomic.LoadInt32(&m.workerCount))

			m.mu.Lock()
			// Пропорционально увеличиваем или уменьшаем количество воркеров
			if queueLen > currentWorkers && currentWorkers < m.maxWorkers {
				additionalWorkers := (queueLen - currentWorkers) / 2
				for i := 0; i < additionalWorkers && currentWorkers < m.maxWorkers; i++ {
					m.addWorker()
					currentWorkers++
				}
				fmt.Printf("Increased workers to %d based on queue length %d\n", currentWorkers, queueLen)
			} else if queueLen < currentWorkers && currentWorkers > m.minWorkers {
				removeWorkers := (currentWorkers - queueLen) / 2
				for i := 0; i < removeWorkers && currentWorkers > m.minWorkers; i++ {
					m.removeWorker()
					currentWorkers--
				}
				fmt.Printf("Decreased workers to %d based on queue length %d\n", currentWorkers, queueLen)
			}
			m.mu.Unlock()
		case <-m.stopChan:
			return // Завершаем работу авто-маппинга при остановке менеджера
		}
	}
}

// Завершение одного воркера
func (m *Manager) removeWorker() {
	select {
	case m.taskQueue <- "stop": // Отправляем команду завершения воркеру
	default:
	}
}

// Добавление задачи в очередь
func (m *Manager) AddTask(task string) {
	m.taskQueue <- task
}

// Останавливает все воркеры и завершает работу
func (m *Manager) Stop() {
	close(m.stopChan)  // Закрываем канал для сигнала остановки
	close(m.taskQueue) // Закрываем канал задач, чтобы завершить всех воркеров
	m.wg.Wait()        // Ждем завершения всех воркеров
}

func main() {
	manager := NewManager(2, 10) // Минимум 2 воркера, максимум 10

	// Добавляем задачи с задержкой
	for i := 0; i < 20; i++ {
		manager.AddTask(fmt.Sprintf("Task %d", i))
		time.Sleep(200 * time.Millisecond)
	}

	manager.Stop() // Останавливаем все воркеры и завершаем работу
	fmt.Println("All tasks completed, exiting program.")
}
