## Руководство по запуску программы
## Запуск через Docker
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

## Запуск линетра
Запустите в терминале
```bash
golangci-lint run
```
## Рубрика проблемы
1. Траблы с запуском докера я запускал докер образы, но поскольку не владею платной подпиской на впн тестировал некоторые фичи через запуск через makefile
2. В апи был сваггер 2.0 поэтому с помощью собранного докер образа для код гена пришлось подкладывать openapi3
