// worker_pool.go

package workerpool

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Структура, представляющая отдельного воркера
type Worker struct {
	id       int32
	taskChan <-chan string
	stopChan chan struct{}
	wg       *sync.WaitGroup
}

// Запуск воркера
func (w *Worker) Start() {
	go func() {
		defer w.wg.Done()
		for {
			select {
			case task, ok := <-w.taskChan:
				if !ok {
					return // Канал задач закрыт, завершаем воркера
				}
				fmt.Printf("Worker %d processing task: %s\n", w.id, task)
				time.Sleep(1 * time.Second) // Имитация обработки задачи
			case <-w.stopChan:
				return // Получен сигнал остановки, завершаем работу воркера
			}
		}
	}()
}

// Остановка воркера
func (w *Worker) Stop() {
	close(w.stopChan)
}

// Структура Manager для управления пулом воркеров
type Manager struct {
	taskQueue   chan string
	stopChan    chan struct{}
	workerCount int32
	maxWorkers  int
	minWorkers  int
	mu          sync.Mutex
	wg          sync.WaitGroup
}

// Конструктор Manager
func NewManager(minWorkers, maxWorkers int) *Manager {
	return &Manager{
		taskQueue:  make(chan string, 100),
		stopChan:   make(chan struct{}),
		minWorkers: minWorkers,
		maxWorkers: maxWorkers,
	}
}

// Метод запуска воркер-пула
func (m *Manager) Start() {
	for i := 0; i < m.minWorkers; i++ {
		m.addWorker()
	}
	go m.manage()
}

// Метод добавления нового воркера
func (m *Manager) addWorker() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if int(m.workerCount) >= m.maxWorkers {
		return
	}

	workerID := atomic.AddInt32(&m.workerCount, 1)
	worker := &Worker{
		id:       workerID,
		taskChan: m.taskQueue,
		stopChan: make(chan struct{}),
		wg:       &m.wg,
	}
	m.wg.Add(1)
	worker.Start()
}

// Метод удаления воркера
func (m *Manager) removeWorker() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if int(m.workerCount) <= m.minWorkers {
		return
	}

	workerID := atomic.AddInt32(&m.workerCount, -1)
	fmt.Printf("Worker %d stopped, total workers: %d\n", workerID, m.workerCount)
}

// Функция автоматического масштабирования воркеров
func (m *Manager) manage() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			queueLen := len(m.taskQueue)
			currentWorkers := int(atomic.LoadInt32(&m.workerCount))

			if queueLen > currentWorkers && currentWorkers < m.maxWorkers {
				m.addWorker()
			} else if queueLen < currentWorkers && currentWorkers > m.minWorkers {
				m.removeWorker()
			}
		case <-m.stopChan:
			return
		}
	}
}

// Добавление задачи в очередь
func (m *Manager) AddTask(task string) {
	m.taskQueue <- task
}

// Останавливает все воркеры и завершает работу
func (m *Manager) Stop() {
	close(m.stopChan)
	close(m.taskQueue)
	m.wg.Wait()
}
