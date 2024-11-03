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
		defer w.wg.Done() // Это единственное место для вызова Done() для данного воркера
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
	workers     map[int32]*Worker
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
		workers:    make(map[int32]*Worker),
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

	// Увеличиваем счетчик воркеров
	workerID := atomic.AddInt32(&m.workerCount, 1)
	// Добавляем в WaitGroup
	m.wg.Add(1)

	worker := &Worker{
		id:       workerID,
		taskChan: m.taskQueue,
		stopChan: make(chan struct{}),
		wg:       &m.wg,
	}

	m.workers[workerID] = worker
	fmt.Printf("Worker %d started, total workers: %d\n", workerID, m.workerCount)
	worker.Start()
}

// Метод удаления воркера
func (m *Manager) removeWorker() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if int(m.workerCount) <= m.minWorkers {
		return
	}

	// Уменьшаем счетчик воркеров
	workerID := atomic.AddInt32(&m.workerCount, -1)

	if worker, exists := m.workers[workerID]; exists {
		worker.Stop()               // Вызываем Stop у воркера
		delete(m.workers, workerID) // Удаляем воркера из map
	}
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
				additionWorkers := (queueLen - currentWorkers) / 2
				for i := 0; i < additionWorkers; i++ {
					m.addWorker()
					currentWorkers++
				}
				fmt.Printf("Increased workers to %d based on queue length %d\n", int(m.workerCount), queueLen)

			} else if queueLen < currentWorkers && currentWorkers > m.minWorkers {
				for currentWorkers != m.minWorkers {
					m.removeWorker()
					currentWorkers--
				}
				fmt.Printf("Decreased workers to %d based on queue length %d\n", int(m.workerCount), queueLen)
			}
		case <-m.stopChan:
			for _, worker := range m.workers {
				worker.Stop()
			}
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
	// Сигнализируем всем воркерам остановиться
	close(m.stopChan)

	// Ожидаем завершения всех воркеров
	m.wg.Wait()

	// Закрываем очередь задач после завершения всех воркеров
	close(m.taskQueue)

	atomic.StoreInt32(&m.workerCount, 0)
}
