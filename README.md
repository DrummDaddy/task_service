# Task ServiceСервис для управления задачами и командами с аутентификацией по JWT, кэшем в Redis, отчётами и метриками Prometheus.
Язык: Go 1.22+
HTTP: chi
DB: MySQL 8.4
Cache: Redis 7
Логи: zap
Метрики: Prometheus (/metrics)
Аутентификация: JWT (Bearer)
Устойчивость: circuit breaker для email-клиента (gobreaker)
Архитектура: Clean Architecture (порты/юзкейсы/адаптеры)
АрхитектураПроект разделён по слоям.
internal/core

ports — доменные интерфейсы (порты): UserRepository, TeamRepository, TaskRepository, ReportsRepository, TasksCache, EmailSender, TokenIssuer и т.д.
usecase — прикладные сценарии (бизнес-логика): AuthUsecase, TeamUseCase, TaskUsecase, ReportUsecase.

internal/repo — адаптеры для MySQL (реализации портов репозиториев).
internal/cache — адаптер Redis (реализация TasksCache).
internal/email — адаптер email-сервиса (реализация EmailSender).
internal/infra/jwt — адаптер для выпуска токенов (реализация TokenIssuer).
internal/app — HTTP-сервер, middleware, композиционный корень (wiring зависимостей).
internal/handler — тонкие HTTP-обработчики, вызывающие use case-ы.
internal/auth — JWT-парсер и middleware аутентификации.
internal/config — загрузка конфигурации через viper (env + опционально config.yaml).
internal/db — подключение MySQL и Redis.
migrations — SQL-миграции (исполняются MySQL при первом старте контейнера).
Принципы:

usecase зависит только от ports (интерфейсов), а не от инфраструктурных пакетов.
адаптеры (repo/cache/email/jwt) реализуют порты.
server.go собирает зависимости и прокидывает их в хендлеры.
# Пример структуры (сокращённо):

cmd/
internal/

app/
auth/
cache/
core/

ports/
usecase/

db/
email/
handler/
infra/

jwt/

repo/
config/

migrations/
go.mod, go.sum
Быстрый старт (Docker Compose)
Убедитесь, что Docker и Docker Compose установлены.

В корне проекта создайте/используйте docker-compose.yml (пример вы, вероятно, уже применили). Важно, чтобы заданы ENV переменные с префиксом TASK.
Минимальный набор переменных:

TASK_MYSQL_DSN="root:rootpass@tcp(mysql:3306)/task_service?parseTime=true&multiStatements=true"
TASK_REDIS_ADDR="redis:6379"
TASK_AUTH_JWTSECRET="dev-secret"

# Запуск:

docker compose up -d

Проверка:

curl http://localhost:8080/healthz → ok
curl http://localhost:8080/api/v1/ping → pong
Примечания:

Для MySQL 8.4 НЕ используйте флаг --default-authentication-plugin=mysql_native_password (он удалён). Уберите его из compose, если был.
При первом старте MySQL применит SQL из каталога ./migrations (если volume пуст).
Локальный запуск (без Docker)
Запустите MySQL и Redis локально (или в контейнерах).

Экспортируйте переменные окружения (viper считывает их с префиксом TASK):

Linux/macOS (bash/zsh):
export TASK_HTTP_ADDR=":8080"
export TASK_MYSQL_DSN="root:rootpass@tcp(127.0.0.1:3306)/task_service?parseTime=true&multiStatements=true"
export TASK_REDIS_ADDR="127.0.0.1:6379"
export TASK_AUTH_JWTSECRET="dev-secret"

Windows (PowerShell):
$env:TASK_HTTP_ADDR=":8080"
$env:TASK_MYSQL_DSN="root:rootpass@tcp(127.0.0.1:3306)/task_service?parseTime=true&multiStatements=true"
$env:TASK_REDIS_ADDR="127.0.0.1:6379"
$env:TASK_AUTH_JWTSECRET="dev-secret"

# Сборка и запуск:

go run ./cmd

Проверка healthz и API — как выше.

Конфигурация
Конфиг читается из:

переменных окружения с префиксом TASK (приоритет),
файла configs/config.yaml (опционально; путь добавлен в viper),
дефолтов в коде (см. internal/config/config.go).

# Ключевые параметры (ENV в скобках):

1. HTTP.addr (TASK_HTTP_ADDR, по умолчанию :8080)
2. MySQL.dsn (TASK_MYSQL_DSN) — обязателен
3. Redis.addr (TASK_REDIS_ADDR, по умолчанию :6379)
4. Redis.db (TASK_REDIS_DB, по умолчанию 0)
5. Auth.jwt_secret (TASK_AUTH_JWTSECRET) — обязателен
6. Auth.access_token_ttl (TASK_AUTH_ACCESS_TOKEN_TTL, по умолчанию 24h)
7. Auth.password_pepper (TASK_AUTH_PASSWORD_PEPPER)
8. Auth.password_hash_cost (TASK_AUTH_PASSWORD_HASH_COST, по умолчанию 12)
9. Cache.tasks_ttl (TASK_CACHE_TASKS_TTL, по умолчанию 5m)
10. RateLimit.per_user_per_minute (TASK_RATE_LIMIT_PER_USER_PER_MINUTE, по умолчанию 100)
11. Email.base_url (TASK_EMAIL_BASE_URL)
12. Email.timeout (TASK_EMAIL_TIMEOUT, по умолчанию 2s)

Если при старте видите panic: jwt secret required — задайте TASK_AUTH_JWTSECRET.
# Миграции

- Поместите SQL-скрипты в ./migrations (минимум 001_init.sql).
 - При старте MySQL из compose, скрипты в docker-entrypoint-initdb.d применяются при пустом data dir.
- Для локальной базы примените SQL вручную или используйте ваш мигратор.

# API
Аутентификация: Bearer JWT (Authorization: Bearer <token>). Открыты только /api/v1/register, /api/v1/login, /api/v1/ping и /healthz.
# Основные endpoints:

- GET /healthz — проверка готовности (MySQL, Redis)
- GET /metrics — метрики Prometheus

# Auth:

- POST /api/v1/register — регистрация {email, password}
- POST /api/v1/login — вход, ответ {access_token, token_type=Bearer}

# Teams (JWT):

- POST /api/v1/teams — создать команду
- GET /api/v1/teams — список команд пользователя
- POST /api/v1/teams/{id}/invite — пригласить пользователя (owner/admin)

# Tasks (JWT):

- POST /api/v1/tasks — создать задачу
- GET /api/v1/tasks?team_id=...&status=...&assignee_id=...&limit=&offset= — список задач
- PUT /api/v1/tasks/{id} — обновить (меняются поля, история пишется)
- GET /api/v1/tasks/{id}/history — история изменений
- POST /api/v1/tasks/{id}/comments — добавить комментарий
- GET /api/v1/tasks/{id}/comments — список комментариев

# Reports (JWT):

- GET /api/v1/reports/team-stats
- GET /api/v1/reports/top-creators
- GET /api/v1/reports/integrity/invalid-assignees?limit=&offset=

- Rate limiting: 429 Too Many Requests при превышении лимита (per user per minute), см. internal/app/middleware_ratelimit.go.

- Логи: zap (dev/prod режим определяется cfg.Env).
- Метрики: /metrics (Prometheus клиент). Middleware собирает счётчики и гистограммы по методу, пути и статусу.

# Тестирование
- Unit-тесты для use case-ов расположены в internal/core/usecase/*_test.go.

- Запуск всех тестов:
go test ./... -v

# Сборка (Docker)
- Если Dockerfile находится в docker/Dockerfile, собирайте с контекстом корня:
docker build -f docker/Dockerfile .

В compose:
yamlservices:
app:
build:
context: .                # корень (рядом с go.mod)
dockerfile: docker/Dockerfile
Важно:

Контекст сборки должен включать go.mod/go.sum. Ошибка COPY go.mod go.sum ./ означает, что контекст неправильный или .dockerignore исключает файлы.
На Windows/OneDrive возможны проблемы с путями. При необходимости перенесите проект в обычный каталог (например, C:\projects\task_service) или используйте WSL.

# Частые проблемы и решения

MySQL 8.4 падает с “unknown variable 'default-authentication-plugin=mysql_native_password'”:

Удалите этот флаг из compose (он больше не поддерживается).
Удалите volume (docker compose down -v) и запустите заново.

panic: jwt secret required:

Установите TASK_AUTH_JWTSECRET.

Redis «unresolved» или healthz падает:

Убедитесь, что используете github.com/redis/go-redis/v9.
В healthz проверяйте if _, err := redisClient.Ping(r.Context()).Result(); err != nil { ... }.

COPY go.mod go.sum при сборке падает:

Проверьте, что контекст сборки — корень проекта, и .dockerignore не исключает go.mod/go.sum.