# TestTaskVk

Использование
====
Основной код находится по пути TestTaskVK/internal/workerpool/worker_pool.go\
\
Чтобы запустить тесты, воспользуйтесь ```go test ./internal/workerpool -v``` (файл с тестами находится по пути TestTaskVK/internal/workerpool/worker_pool_test.go)\
\
Чтобы запустить пример выполнение, воспользуйтесь ```go run cmd/main/main.go``` (файл с примером находится по пути TestTaskVK/cmd/main/main.go)
## Описание проекта

Пул воркеров получает входные данные (строки) через канал, после чего воркеры обрабатывают их. Каждый воркер читает задачи из канала и обрабатывает их, выводя на экран свой ID и данные задачи. Количество активных воркеров может динамически изменяться в зависимости от нагрузки — воркеры добавляются или удаляются по необходимости.


Стратегия менеджера worker pool
===
Для менеджера WorkerPool в этом проекте  была использована стратегия на основе пропорционального увелечения воркеров (увеличиваем количество воркеров пропорционально числу задач в очереди), тк это гибкий подход, который может эффективно распределять нагрузку. Проблема резкого роста воркеров была решена с помощью ограничения сверху и снизу количества воркеров.

Менеджер удаляет воркеров, когда их количество больше, чем количество задач, тк в таком случае воркеры потребляют ресурсы и не выполняют работы.
 
