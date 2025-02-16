## Рубрика проблемы
1. Докер сейчас очень трудно запустить на терриритории России поэтому некоторые фичи и ручки проверял запуском через makefile
2. В апи был сваггер 2.0 поэтому с помощью собранного докер образа для код гена пришлось подкладывать openapi3
3. Выполнил программу по чистой архитектуре, но поскольку выбрал интерфейс для работы с БД на уровне репозитория базы данных, то тесты получились достаточно обьемными и пришлось мокать БД
4. В принципе достаточно сложно было спроектировать архитектуру проекта, но в итоге построив схемы и диаграммы в draw.io я преодолел эту проблему
## Структура проекта
```
.
├── Dockerfile
├── Dockerfile.oapi-codegen
├── Makefile
├── README.md
├── build
├── cmd
│   └── main.go
├── docker-compose.yml
├── go.mod
├── go.sum
├── integration
│   └── integration_test.go
├── internal
│   ├── api
│   │   ├── handlers.go
│   │   └── openapi.gen.go
│   ├── config
│   │   └── config.go
│   ├── db
│   │   ├── db.go
│   │   ├── migration.go
│   │   └── queries.go
│   ├── middleware
│   │   └── middleware.go
│   ├── models
│   │   └── models.go
│   └── service
│       ├── auth.go
│       ├── auth_test.go
│       ├── shop.go
│       └── shop_test.go
├── migrations
│   └── 0001_init.sql
├── openapi.json
├── openapi3.json
└── pkg
    └── logger.go

```
## Руководство по запуску программы
## Запуск через Docker
В корне проекта соберите файл Dockerfile.oapi-codegen
```bash
docker build -f Dockerfile.oapi-codegen -t local/oapi-codegen 
```
В корневой директории проекта выполните команду:
```bash
docker-compose up --build
```
Для проверки корректности работы программы можно запусить
```bash
docker-compose logs
```
Чтобы остановить и удалить контейнеры, выполните:
```bash
docker-compose down
```
## Запуск через Makefile

Поскольку утилиту Docker в России стало использовать проблематично предлагаю альтернативный и более удобный способ. 

В консоли выполните следующее


В корне проекта соберите файл Dockerfile.oapi-codegen
```bash
docker build -f Dockerfile.oapi-codegen -t local/oapi-codegen 
```
```bash
make db-up
```


```bash
make migrate
```
убедитесь что PostrgreSQL выключен иначе они будут слушать один порт

```bash
make all
```

Тесты запустятся автоматически ,но можно запустить и руками 

Юнит тесты 
```bash
make test
```
Интеграционные тесты
```bash
go test -v ./integration
```

## Запуск линтера
Запустите в терминале
```bash
golangci-lint run
```
