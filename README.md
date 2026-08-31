# log-parser

Утилита на Go для анализа лог-файлов сервисов: находит запросы (`request_id`),
в цепочке которых произошла ошибка или предупреждение, и выгружает отчёт в JSON.

## Как это работает

1. **Сканирование** (`internal/scanner`) — рекурсивно обходит директорию с логами
   и собирает пути ко всем файлам с расширением `.log`.
2. **Чтение и парсинг** (`internal/parser`, `internal/processor`) — каждый файл
   читается построчно, строки парсятся регуляркой в структуру `LogEntry`
   (timestamp, level, service, message, request_id, user_id). Файлы
   обрабатываются параллельно пулом воркеров (`ProcessFilesConcurrently`,
   размер пула задан в `main.go`).
3. **Корреляция** (`processor.CorrelateRequests`) — все записи группируются
   по `request_id`, чтобы восстановить цепочку событий одного запроса
   через разные сервисы.
4. **Детект ошибок** (`processor.DetectFailedRequests`, `FindFirstFailure`) —
   для каждого `request_id` проверяется, есть ли в цепочке запись с уровнем
   `ERROR` или `WARNING`. Если да — находится самая ранняя такая запись
   (первая точка отказа) и вся временная шкала запроса сортируется по времени.
5. **Отчёт** (`internal/reporter`) — результат (сколько всего записей
   обработано, сколько запросов упало, время обработки, и по каждому
   упавшему запросу: где именно упал, с каким сообщением, полная временная
   шкала) сохраняется в JSON-файл.

Программа корректно завершается по Ctrl+C (SIGINT) — контекст отменяется,
воркеры останавливаются.

### Формат строки лога

```
2023-12-25T14:30:15.123Z [INFO] user-service: User authenticated, request_id=req_abc123, user_id=12345
```

Уровень — один из `INFO`, `WARNING`, `ERROR`. Строки другого формата
пропускаются, не прерывая обработку файла.

## Запуск

Требуется Go 1.26+.

```bash
go run . -input-dir logs -output-file results.json
```

Флаги:

| Флаг | По умолчанию | Описание |
|---|---|---|
| `-input-dir` | `logs` | Директория с `.log` файлами (ищет рекурсивно) |
| `-output-file` | `results.json` | Куда записать JSON-отчёт |

Если `-input-dir` не существует — программа завершится с ошибкой.

### Сборка бинарника

```bash
go build -o bin/log-parser .
./bin/log-parser -input-dir logs -output-file results.json
```

## Структура проекта

```
main.go                    точка входа: связывает cli → scanner → processor → reporter
internal/cli/               разбор флагов командной строки
internal/scanner/           поиск .log файлов в директории
internal/parser/            парсинг строки лога в LogEntry, LogEntry.IsError()
internal/processor/         чтение файлов, конкурентная обработка, корреляция по request_id
internal/reporter/          структура отчёта и запись в JSON
```

## Пример отчёта

```json
{
  "total_entries_processed": 396,
  "failed_requests_found": 12,
  "processing_time_seconds": 0.004,
  "failed_requests": [
    {
      "request_id": "req_abc123",
      "failing_service": "payment-service",
      "error_message": "Payment declined",
      "timeline": [
        "2023-12-25T14:30:15.123Z [INFO] user-service: User authenticated",
        "2023-12-25T14:30:16.900Z [ERROR] payment-service: Payment declined"
      ]
    }
  ]
}
```
