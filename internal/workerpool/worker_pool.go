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

type WorkerPool struct {
	workers     map[int32]*Worker
	taskQueue   chan string
	stopChan    chan struct{}
	workerCount int32
	maxWorkers  int
	minWorkers  int
	mu          *sync.Mutex
	wg          *sync.WaitGroup
}

func NewWorkerPool(minWorkers, maxWorkers int) *WorkerPool {
	return &WorkerPool{
		workers:    make(map[int32]*Worker),
		taskQueue:  make(chan string, 100),
		stopChan:   make(chan struct{}),
		minWorkers: minWorkers,
		maxWorkers: maxWorkers,
		mu:         &sync.Mutex{},
		wg:         &sync.WaitGroup{},
	}
}

// запускаем воркер пул
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.minWorkers; i++ {
		wp.addWorker()
	}
	go wp.manage()
}

// добавляем воркера
func (wp *WorkerPool) addWorker() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if int(wp.workerCount) >= wp.maxWorkers { // важно: менеджер постоянно пытается добавить воркера, здесь мы не даем ему добавить больше нужного
		return
	}

	// Увеличиваем счетчик воркеров
	workerID := atomic.AddInt32(&wp.workerCount, 1)
	wp.wg.Add(1)

	worker := &Worker{
		id:       workerID,
		taskChan: wp.taskQueue,
		stopChan: make(chan struct{}),
		wg:       wp.wg,
	}

	wp.workers[workerID] = worker
	fmt.Printf("Worker %d started, total workers: %d\n", workerID, wp.workerCount)
	worker.Start()
}

// удаляем воркера
func (wp *WorkerPool) removeWorker() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if int(wp.workerCount) <= wp.minWorkers { // важно: менеджер постоянно пытается удалить воркера, здесь мы не даем ему удалить больше нужного
		return
	}

	// Находим первого попавшегося воркера для удаления
	for id, worker := range wp.workers {
		worker.Stop()
		delete(wp.workers, id)
		break
	}

	// Уменьшаем счетчик воркеров
	workerID := atomic.AddInt32(&wp.workerCount, -1)
	fmt.Printf("Worker %d stopped, total workers: %d\n", workerID, wp.workerCount)
}

// тут самое интересное: менеджер для динамически изменяемого воркер пула
func (wp *WorkerPool) manage() {
	// будем проверять очередь задач и количество рабочих воркеров раз в какоето время
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			queueLen := len(wp.taskQueue)
			currentWorkers := int(atomic.LoadInt32(&wp.workerCount))
			// тот случай, когда нам надо увеличить количество воркеров
			if queueLen > currentWorkers && currentWorkers < wp.maxWorkers {
				additionWorkers := (queueLen - currentWorkers) / 2 // на какое количество увеличиваем
				for i := 0; i < additionWorkers; i++ {
					wp.addWorker()
					currentWorkers++
				}
				fmt.Printf("Increased workers to %d based on queue length %d\n", int(wp.workerCount), queueLen)
				// когда надо уменьшить
			} else if queueLen < currentWorkers && currentWorkers > wp.minWorkers {
				for currentWorkers != wp.minWorkers { // удаляем сразу все лишние воркеры
					wp.removeWorker()
					currentWorkers--
				}
				fmt.Printf("Decreased workers to %d based on queue length %d\n", int(wp.workerCount), queueLen)
			}
		case <-wp.stopChan: // если поступил сигнал об остановки воркер пула
			for _, worker := range wp.workers {
				worker.Stop()
			}
			return
		}
	}
}

// добавляем задачу в очередь
func (wp *WorkerPool) AddTask(task string) {
	wp.taskQueue <- task
}

// останавливаем воркер пул
func (wp *WorkerPool) Stop() {
	close(wp.stopChan)
	wp.wg.Wait()
	close(wp.taskQueue)
	atomic.StoreInt32(&wp.workerCount, 0)
}
