# 🚀 GopherWallet Monorepo: Финтех-платформа электронных кошельков (учебный проект)

Это распределенная событийно-ориентированная (Event-Driven) платформа электронных кошельков, написанная на Go 1.26. 

Проект спроектирован по правилам **Clean Architecture** с использованием монорепозитория (**Go Workspaces**). Вся инфраструктура (PostgreSQL 18, Kafka 8.2.2, Redis 8.0) разворачивается одной командой.

---

## 🗺️ Пояснения для PHP-разработчиков, переучивающихся на Go :)

1. **Нет виртуальной машины:** Здесь нет `php-fpm` или Nginx. Каждый сервис компилируется в бинарник и сам является долгоживущим HTTP/gRPC сервером или демоном. 
2. **Пул соединений «из коробки»:** Забудьте про постоянное переподключение к БД на каждый чих или pgbouncer. Приложение держит постоянный пул соединений (`pgxpool`).
3. **Асинхронность и Очереди:** `Wallet Service` блокирует баланс транзакцией в Postgres, меняет его и мгновенно выплевывает JSON-событие в Kafka. Он освобождает память всего за 20-30мс. Сервис `Notification` и другие (будущие) сервисы параллельно и асинхронно ловят это событие из Kafka.
4. **Dependency Injection:** Вместо тяжелой магии Laravel Service Container или Symfony DI, в Go используется утилита **Google Wire**. Она собирает зависимости (`Repository -> UseCase -> Handler`) на этапе компиляции, генерируя чистый Go-код без медленной рефлексии в рантайме.

---

## 🛠️ Как развернуть и протестировать локально

### 1. Подготовка окружения

#### Общее требование: Установка Go
На вашем компьютере должен быть установлен **Go 1.26** или выше. 
* Проверить версию: `go version`
* Если языка нет, скачайте официальный дистрибутив с [golang.org/dl](https://golang.org) (для Ubuntu рекомендуется ставить через тарбол, а не через устаревший `apt-get`, чтобы не получить древнюю версию 1.18).

#### Вариант А: Если у вас Windows 10/11 + WSL 2 (Ubuntu)
Убедитесь, что у вас запущен Docker Desktop в Windows и включена интеграция с WSL (Settings -> Resources -> WSL integration -> включить тумблер на вашей Ubuntu).

#### Вариант Б: Если у вас чистый Linux (Ubuntu)
Убедитесь, что у вас установлены `docker` и `docker-compose` (плагин `docker-compose-plugin`). Ваши текущие системные пользователи должны быть добавлены в группу `docker`, чтобы команды выполнялись без `sudo`:
```bash
sudo usermod -aG docker \$USER
# Перезапустите сессию терминала, чтобы права обновились
```

### 2. Запуск инфраструктуры
Из корня монорепозитория поднимите контейнеры (PostgreSQL 18, Kafka 8.2.2, Redis 8.0):
```bash
docker compose up -d
```

### 3. Инициализация рабочего пространства Go
В корне репозитория (файлы `go.work` внесены в `.gitignore`, поэтому создаем рабочую зону локально):
```bash
go work init ./wallet-service ./notification-service
```

### 4. Накат миграций базы данных
В Go мы не используем тяжелые ORM. Схема данных и история транзакций (Double-Entry Bookkeeping) версионируются через утилиту `goose`.
```bash
# Установка утилиты goose в WSL
go install github.com/pressly/goose/v3/cmd/goose@latest

# Накат миграции (выполнять из папки wallet-service)
goose -dir migrations postgres "postgres://wallet_user:wallet_password@localhost:5432/wallet_db?sslmode=disable" up
```
*(После наката в Postgres автоматически создастся тестовый пользователь `user_123` с балансом 1000.00 руб).*

### 5. Создание топика в Kafka
Поскольку Kafka чистая, создадим топик `transactions` вручную внутри контейнера:
```bash
docker exec -it wallet_kafka kafka-topics --create --topic transactions --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
```

### 6. Запуск микросервисов (в разных терминалах WSL)
* **Запуск Wallet Service (порт 8081):**
  ```bash
  cd wallet-service
  go run ./cmd/app
  ```
* **Запуск Notification Service (воркер):**
  ```bash
  cd notification-service
  go run main.go
  ```

### 7. Тестирование через cURL
Откройте третий терминал и отправьте POST-запрос на пополнение кошелька:
```bash
curl -X POST http://localhost:8081/api/v1/deposit \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user_123", "amount": 50000}'
```
В терминале `Notification Service` вы мгновенно увидите лог перехваченного пуша, а транзакция запишется в Redis-кэш истории!

Проверьте логику валидации баланса, попытавшись списать больше доступного:
```bash
curl -X POST http://localhost:8081/api/v1/debit \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user_123", "amount": 999999}'
```

---

## 🧱 Инструкция: Как создать новый микросервис в этом проекте

Например, мы хотим добавить новый микросервис `anti-fraud-service`:

1. **Создайте ветку** в Git от `main` (например, `feature/anti-fraud`).
2. **Создайте новую папку** в корне монорепозитория: `mkdir anti-fraud-service`.
3. **Инициализируйте Go-модуль** внутри этой папки через вложенный путь (Sub-module), чтобы сохранить связь с монорепозиторием:
   ```bash
   cd anti-fraud-service
   go mod init github.com/aimv/gopher-wallet-monorepo/anti-fraud-service
   ```
4. **Обновите свой локальный `go.work`** в корне монорепозитория, чтобы ваша IDE (VS Code / GoLand) увидела новый сервис:
   ```bash
   cd ..
   go work use ./anti-fraud-service
   ```
5. Подключитесь к Kafka (адрес `localhost:9092`, топик `transactions`). Задайте уникальный `GroupID` для своего сервиса в конфиге ридера, чтобы читать поток событий независимо от `Notification Service`.
