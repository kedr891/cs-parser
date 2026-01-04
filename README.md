# CS:GO Skins Parser

Микросервис для парсинга и отслеживания цен на скины CS:GO с поддержкой шардирования PostgreSQL.

## Архитектура

```
internal/
├── api/                    # gRPC/REST API handlers
├── bootstrap/              # Инициализация компонентов
├── consumer/               # Kafka consumers
├── models/                 # Модели данных
├── pb/                     # Protobuf файлы
├── services/               # Бизнес-логика
│   ├── skinService/       # + тесты + моки
│   ├── analyticsService/  # + моки
│   └── processors/
└── storage/
    ├── db/                # PostgreSQL клиент
    ├── pgstorage/         # Репозитории (squirrel)
    ├── redis/             # Redis клиент
    └── sharding/          # Логика шардирования
```

## Технологии

- **Go 1.23**
- **gRPC + gRPC-Gateway** - API
- **PostgreSQL 18** - БД с шардированием по weapon
- **Redis 8** - кеширование
- **Kafka** - очередь сообщений
- **Squirrel** - SQL query builder
- **Mockery** - генерация моков для тестов

## Шардирование

Данные распределяются по 3 шардам:
- **Shard 0 (Pistols)**: Desert Eagle, Glock-18, USP-S, P250, Five-SeveN, Tec-9, CZ75-Auto, Dual Berettas, P2000, R8 Revolver
- **Shard 1 (Rifles)**: AK-47, M4A4, M4A1-S, AWP, SSG 08, SCAR-20, G3SG1, AUG, SG 553, Galil AR, FAMAS
- **Shard 2 (Other)**: Knives, Gloves, Stickers, Cases, Keys, и остальное

## Запуск

### Локальная разработка

```powershell
# 1. Запустить инфраструктуру
docker-compose --profile sharding up -d postgres_shard_pistols postgres_shard_rifles postgres_shard_other redis kafka

# 2. Собрать API
go build -o bin/api.exe ./cmd/api

# 3. Запустить API
$env:configPath="config.local.yaml"
.\bin\api.exe
```

### Docker (полный стек)

```powershell
# Собрать образы
docker-compose build

# Запустить с шардированием
docker-compose --profile sharding up -d

# Просмотр логов
docker-compose --profile sharding logs -f api

# Остановить
docker-compose --profile sharding down

# Очистить volumes
docker-compose --profile sharding down -v
```

## API Endpoints

### REST
- `POST /api/v1/skins` - создать скин
- `GET /api/v1/skins` - список скинов (с фильтрами)
- `GET /api/v1/skins/{slug}` - детали скина
- `GET /api/v1/skins/search` - поиск скинов
- `GET /api/v1/skins/popular` - популярные скины
- `GET /api/v1/analytics/trending` - трендовые скины
- `GET /api/v1/analytics/top-gainers` - топ растущих
- `GET /api/v1/analytics/top-losers` - топ падающих
- `GET /api/v1/analytics/market-overview` - обзор рынка

### Swagger UI
- http://localhost:8080/docs/index.html

### gRPC
- localhost:50051

## Тестирование

```powershell
# Все тесты
go test -v ./...

# Только skinService
go test -v ./internal/services/skinService/...

# С покрытием
go test -cover ./...
```

## Конфигурация

- `config.yaml` - для Docker (хосты: postgres_shard_*, redis, kafka)
- `config.local.yaml` - для локальной разработки (localhost)

## Makefile команды

```bash
make generate      # Генерация proto файлов
make build-api     # Сборка API
make test          # Запуск тестов
make docker-up     # Запуск Docker с шардированием
make docker-down   # Остановка Docker
make docker-clean  # Очистка Docker volumes
make docker-logs   # Просмотр логов API
```

## Пример создания скина

```powershell
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/skins" `
  -Method POST `
  -Headers @{"Content-Type"="application/json"} `
  -Body '{
    "market_hash_name": "AK-47 | Redline",
    "name": "Redline",
    "weapon": "AK-47",
    "quality": "Field-Tested",
    "rarity": "Classified",
    "current_price": 15.50,
    "currency": "USD",
    "image_url": "https://example.com/img.jpg"
  }'
```

## Структура БД

### Таблицы
- `skins` - основная таблица скинов
- `price_history` - история цен

### Индексы
- `skins_slug_key` - уникальный slug
- `skins_weapon_idx` - для шардирования
- `skins_price_idx` - для фильтрации по цене
- `skins_volume_idx` - для популярных скинов

## Мониторинг

- **AKHQ (Kafka UI)**: http://localhost:8081
- **PostgreSQL**: localhost:5433, 5434, 5435
- **Redis**: localhost:6379
- **Kafka**: localhost:9092

