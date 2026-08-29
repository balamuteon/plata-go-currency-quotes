# Plata Currency Quotes

HTTP-сервис асинхронного обновления валютных котировок на Go и `chi`.

Клиент создает задание на обновление котировки, сервис кладет его в устойчивую очередь в PostgreSQL, worker получает актуальный курс во внешнем API и сохраняет результат. Повторные запросы защищаются обязательным `Idempotency-Key`.

Поддерживаемые валюты: `USD`, `EUR`, `MXN`.
Источник данных: [Frankfurter API v2](https://frankfurter.dev/).

## Быстрый старт

Проект использует `.env` файл для конфигурации. Файл `.env.example` предоставлен в качестве примера. Убедитесь, что у вас есть файл `.env` с необходимыми переменными.

```bash
cp .env.example .env
```

Если Goose CLI еще не установлен, установите его перед запуском:

```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.26.0
```

### Запуск окружения и приложения

```bash
make up
```

`make up` поднимает PostgreSQL, применяет миграции через Goose и запускает приложение.
Makefile сам подтянет `GOOSE_DRIVER`, `GOOSE_DBSTRING` и `GOOSE_MIGRATION_DIR` из `.env`.

Проверить, что сервис отвечает:

```bash
curl http://localhost:8080/ping
```

## Тестирование

#### `make test-unit` - Запускает unit-тесты.

#### `make test-integ` - Запускает интеграционные тесты репозитория.

#### `make test` - Запускает unit и integration тесты.

#### `make cover` - Создает отчет о покрытии по unit и integration тестам.

#### `make cover-html` - Открывает HTML-отчет о покрытии.

#### `make lint` - Запускает `golangci-lint`.

## API

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/v1/quote-updates` | Создать фоновое обновление котировки |
| `GET` | `/v1/quote-updates/{update_id}` | Получить статус и результат обновления |
| `GET` | `/v1/quotes/latest?pair=EUR/MXN` | Получить последнюю успешную котировку |
| `GET` | `/ping` | Проверить простой ping |
| `HEAD` | `/healthcheck` | Проверить, что сервис жив |

Создать обновление:

```bash
curl -i \
  -X POST http://localhost:8080/v1/quote-updates \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-eur-mxn-1' \
  -d '{"pair":"EUR/MXN"}'
```

Получить результат:

```bash
curl http://localhost:8080/v1/quote-updates/<update_id>
```

Получить последнюю котировку:

```bash
curl 'http://localhost:8080/v1/quotes/latest?pair=EUR/MXN'
```

Поле `updated_at` в ответах с котировкой означает время, когда наш сервис сохранил последнее успешное обновление цены.

OpenAPI-спецификация лежит в `api/openapi.yaml`.

## Команды

```bash
make up               # поднять PostgreSQL, применить миграции и запустить приложение
make down             # остановить Docker Compose
make fmt              # отформатировать код
make lint             # запустить golangci-lint
make test-unit        # unit-тесты
make test-integ       # интеграционные тесты
make test             # все тесты
make cover            # отчет о покрытии по unit и integration тестам
make cover-html       # HTML-отчет о покрытии
make build            # собрать бинарник
```

## Конфигурация

Все основные переменные перечислены в `.env.example`.

Для локального запуска с хоста в `.env.example` указан `POSTGRES_HOST=localhost`. Docker Compose для контейнера приложения переопределяет `POSTGRES_HOST=postgres`, чтобы приложение внутри Docker-сети подключалось к контейнеру PostgreSQL. Сам `.env` исключен из Git и Docker build context. В production значения должны приходить из среды деплоя или системы управления секретами.

Ключевые переменные:

| Переменная | Описание |
|---|---|
| `POSTGRES_HOST` | хост PostgreSQL для приложения |
| `POSTGRES_PORT` | порт PostgreSQL |
| `POSTGRES_USER` | пользователь PostgreSQL |
| `POSTGRES_PASSWORD` | пароль PostgreSQL |
| `POSTGRES_DB` | база PostgreSQL |
| `POSTGRES_SSLMODE` | режим SSL для PostgreSQL |
| `HTTP_PORT` | порт HTTP-сервера |
| `QUOTE_PROVIDER_BASE_URL` | URL внешнего API котировок |
| `WORKER_COUNT` | число фоновых workers |
| `WORKER_MAX_ATTEMPTS` | максимальное число попыток обработки задания |
| `SHUTDOWN_TIMEOUT` | timeout graceful shutdown |

## Структура

```text
cmd/quotes                    точка входа и сборка зависимостей
internal/config               конфигурация
internal/domain               доменные модели и ошибки
internal/service/quote        сценарии работы с котировками
internal/handler              HTTP handlers
internal/repository/quote     PostgreSQL repository и очередь заданий
internal/gateway/frankfurter  клиент внешнего API
internal/worker/quote         фоновая обработка заданий
internal/router               chi router и middleware
internal/server               HTTP server
internal/app                  lifecycle приложения
pkg/db                        подключение к PostgreSQL
pkg/observability             logger и logging middleware
pkg/testdb                    PostgreSQL testcontainer для интеграционных тестов
migrations                    Goose-миграции
api                           OpenAPI-контракт
```

## Детали реализации

Очередь заданий хранится в PostgreSQL. Workers забирают задания через `FOR UPDATE SKIP LOCKED`, используют lease и retry, поэтому задания не теряются при падении процесса.

Идемпотентность реализована через обязательный заголовок `Idempotency-Key` и уникальное ограничение в базе. Повтор того же запроса возвращает исходный `update_id`, а попытка использовать тот же ключ для другой валютной пары возвращает `409 Conflict`.
