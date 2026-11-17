# URL Shortener

Сервис сокращения ссылок на Go. Поддерживает создание, обновление, удаление ссылок и редирект по alias.

## Стек

Go, chi, PostgreSQL, pgx, slog, cleanenv. Тесты: testify и mockery.

## API

| Метод  | Путь       | Назначение                    | Basic Auth |
| ------ | ---------- | ----------------------------- | ---------- |
| POST   | `/save`    | Создать ссылку                | Да         |
| GET    | `/{alias}` | Перейти по ссылке             | Нет        |
| PUT    | `/{alias}` | Изменить исходный URL         | Да         |
| DELETE | `/{alias}` | Удалить ссылку                | Да         |
| GET    | `/health`  | Проверить доступность сервиса | Нет        |

Создание ссылки:

```json
{
    "url": "https://example.com",
    "alias": "example"
}
```

Если `alias` отсутствует, сервис генерирует его автоматически. Пустой alias возвращает `400`.

Обновление ссылки:

```json
{
    "url": "https://example.org"
}
```

Успешное создание возвращает `201`, редирект - `302`, обновление и удаление - `204`. Удаление отсутствующей ссылки также возвращает `204`.

## Конфигурация

Настройки сервера находятся в `config/dev.yaml`. Пароли и параметры доступа задаются через `.env`.

Создай `.env` в корне проекта:

```dotenv
CONFIG_PATH=config/dev.yaml

POSTGRES_DB=url_shortener
POSTGRES_USER=shortener
POSTGRES_PASSWORD=local_db_password

API_USERNAME=admin
API_PASSWORD=local_api_password
```

Это пример для локального запуска. Файл `.env` не добавляется в Git; образец переменных хранится в `.env.example`.

## Запуск

Создай `.env` в корне проекта:

```dotenv
CONFIG_PATH=config/dev.yaml
POSTGRES_DB=url_shortener
POSTGRES_USER=shortener
POSTGRES_PASSWORD=local_db_password
API_USERNAME=admin
API_PASSWORD=local_api_password
```

Запусти PostgreSQL через Docker Compose, затем приложение:

````bash
docker compose up -d
go run ./cmd/url-shortener

## Тесты

```bash
go test -v -count=1 ./...
````

Юнит-тесты проверяют service и handler. Интеграционные тесты проверяют HTTP-сценарии через роутер с авторизацией и моком repository.

После изменения интерфейсов перегенерируй моки:

```bash
go tool mockery --config .mockery.yaml
```
