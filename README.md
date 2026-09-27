
REST API управления задачами команд на Go, MySQL и Redis.

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

API доступен на `http://localhost:8080`, метрики Prometheus — на `/metrics`. Миграции из `migrations/` автоматически применяются MySQL при создании нового volume.

## Конфигурация

Основные переменные: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `REDIS_ADDR`, `REDIS_PASSWORD`, `JWT_SECRET`, `SERVER_PORT`, `RATE_LIMIT`. Значения для локального запуска приведены в `.env.example`; секреты для production необходимо заменить.

## API

Полная схема запросов, ответов и ошибок находится в `openapi.yaml`.

```bash
# Регистрация
curl -X POST http://localhost:8080/api/v1/register -H 'Content-Type: application/json' \
  -d '{"email":"owner@example.com","name":"Owner","password":"password123"}'

# Список задач с фильтрами
curl 'http://localhost:8080/api/v1/tasks?team_id=1&status=todo&limit=20&offset=0' \
  -H 'Authorization: Bearer <token>'

# Обновление с optimistic locking
curl -X PUT http://localhost:8080/api/v1/tasks/1 -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' -d '{"status":"done","version":1}'

# Отчет команды (owner/admin)
curl http://localhost:8080/api/v1/teams/1/stats -H 'Authorization: Bearer <token>'
```

`version` обязателен при обновлении. Если задача уже была изменена другим запросом, API возвращает `409 Conflict`; клиент должен перечитать задачу и повторить изменение с актуальной версией.

Списки задач кешируются в Redis на 5 минут с учетом `team_id`, фильтров, `limit` и `offset`. После создания или изменения задачи кеши команды инвалидируются.

## Тесты

Для интеграционных тестов нужен запущенный Docker daemon: тесты самостоятельно создают временные контейнеры MySQL и Redis.

```bash
go test ./...
go vet ./...
docker compose config --quiet
```
