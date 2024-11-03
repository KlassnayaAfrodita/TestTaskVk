package workerpool

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Worker struct {
	id       int32
	taskChan <-chan string
	stopChan chan struct{}
	wg       *sync.WaitGroup
}

// Запуск воркера
func (w *Worker) Start() {
	go func() {
		defer w.wg.Done() // важно: Done() вызывается только в этом месте
		for {
			select {
			case task, ok := <-w.taskChan:
				if !ok { // Канал задач закрыт
					return
				}
				fmt.Printf("Worker %d processing task: %s\n", w.id, task)
				time.Sleep(1 * time.Second) // Имитируем какуюлибо работу
			case <-w.stopChan: // получаем сигнал о закрытиии
				return
			}
		}
	}()
}

// Остановка воркера
func (w *Worker) Stop() {
	close(w.stopChan)
}

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

func NewManager(minWorkers, maxWorkers int) *Manager {
	return &Manager{
		workers:    make(map[int32]*Worker),
		taskQueue:  make(chan string, 100),
		stopChan:   make(chan struct{}),
		minWorkers: minWorkers,
		maxWorkers: maxWorkers,
	}
}

// запускаем воркер пул
func (m *Manager) Start() {
	for i := 0; i < m.minWorkers; i++ {
		m.addWorker()
	}
	go m.manage()
}

// добавляем воркера
func (m *Manager) addWorker() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if int(m.workerCount) >= m.maxWorkers { // важно: менеджер постоянно пытается добавить воркера, здесь мы не даем ему добавить больше нужного
		return
	}

	// Увеличиваем счетчик воркеров
	workerID := atomic.AddInt32(&m.workerCount, 1)
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

// удаляем воркера
func (m *Manager) removeWorker() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if int(m.workerCount) <= m.minWorkers { // важно: менеджер постоянно пытается удалить воркера, здесь мы не даем ему удалить больше нужного
		return
	}

	if worker, exists := m.workers[m.workerCount]; exists { // получаем последнего воркера
		worker.Stop()
		delete(m.workers, m.workerCount)
	}

	// Уменьшаем счетчик воркеров
	workerID := atomic.AddInt32(&m.workerCount, -1)
	fmt.Printf("Worker %d stopped, total workers: %d\n", workerID, m.workerCount)
}

// тут самое интересное: менеджер для динамически изменяемого воркер пула
func (m *Manager) manage() {
	// будем проверять очередь задач и количество рабочих воркеров раз в какоето время
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			queueLen := len(m.taskQueue)
			currentWorkers := int(atomic.LoadInt32(&m.workerCount))
			// тот случай, когда нам надо увеличить количество воркеров
			if queueLen > currentWorkers && currentWorkers < m.maxWorkers {
				additionWorkers := (queueLen - currentWorkers) / 2 // на какое количество увеличиваем
				for i := 0; i < additionWorkers; i++ {
					m.addWorker()
					currentWorkers++
				}
				fmt.Printf("Increased workers to %d based on queue length %d\n", int(m.workerCount), queueLen)
				// когда надо уменьшить
			} else if queueLen < currentWorkers && currentWorkers > m.minWorkers {
				for currentWorkers != m.minWorkers { // удаляем сразу все лишние воркеры
					m.removeWorker()
					currentWorkers--
				}
				fmt.Printf("Decreased workers to %d based on queue length %d\n", int(m.workerCount), queueLen)
			}
		case <-m.stopChan: // если поступил сигнал об остановки воркер пула
			for _, worker := range m.workers {
				worker.Stop()
			}
			return
		}
	}
}

// добавляем задачу в очередь
func (m *Manager) AddTask(task string) {
	m.taskQueue <- task
}

// останавливаем воркер пул
func (m *Manager) Stop() {
	close(m.stopChan)
	m.wg.Wait()
	close(m.taskQueue)
	atomic.StoreInt32(&m.workerCount, 0)
}
