package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	// Импортируем сгенерированный gRPC-пакет из соседнего модуля
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/pkg/walletgrpc"
)

// Структура события должна строго совпадать с контрактом из Wallet Service
type TransactionEvent struct {
	UserID string `json:"user_id"`
	Amount int64  `json:"amount"`
	Type   string `json:"type"` // 'DEPOSIT' или 'DEBIT'
}

func main() {
	// Создаем корневой контекст, который мы отменим при получении сигналов от ОС
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// === Инициализируем быстрый кэш Redis 8.0
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	// Обязательно пингуем Redis при старте
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[Notification] Failed to connect to Redis: %v", err)
	}
	fmt.Println("[Notification] Connected to Redis successfully!")

	// === Инициализируем ПОСТОЯННОЕ gRPC-соединение (клиент) к Wallet Service
	// credentials.NewInsecure() отключает TLS/SSL шифрование для локальной разработки
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[Notification] Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	// Создаем gRPC-клиент из сгенерированного пакета
	grpcClient := walletgrpc.NewWalletServiceClient(conn)
	fmt.Println("[Notification] gRPC Client successfully connected to Wallet Service (port 50051)")

	// === Инициализируем Kafka Consumer (Reader)
	// Задаем GroupID, чтобы Kafka помнила, где остановился этот сервис
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{"localhost:9092"},
		Topic:          "transactions",
		GroupID:        "notification-consumer-group",
		CommitInterval: time.Second, // Автоматически коммитить прочитанные сообщения раз в секунду
	})
	defer reader.Close()
	fmt.Println("[Notification] Listening to Kafka topic 'transactions'...")

	// === Настраиваем Graceful Shutdown для воркера
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n[Notification] Shutdown signal received. Stopping worker gracefully...")
		cancel() // Отменяем контекст, чтобы прервать бесконечный цикл чтения
	}()

	// === Основной бесконечный цикл обработки событий
	for {
		// === ReadMessage блокирует поток и ждет появления нового сообщения в Kafka
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			// Если контекст был отменен через Ctrl+C — выходим из цикла без ошибок
			if ctx.Err() != nil {
				break
			}
			log.Printf("[Notification Error] Failed to read message: %v", err)
			continue
		}

		// Десериализуем JSON-событие
		var event TransactionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[Notification Error] Failed to unmarshal JSON: %v", err)
			continue
		}

		// === Делаем синхронный gRPC-запрос в Wallet Service за балансом!
		// Для сетевого запроса ВСЕГДА создаем контекст с коротким таймаутом (2 секунды)
		grpcCtx, grpcCancel := context.WithTimeout(context.Background(), 2*time.Second)
		
		walletData, err := grpcClient.GetBalance(grpcCtx, &walletgrpc.GetBalanceRequest{
			UserId: event.UserID,
		})
		grpcCancel() // Освобождаем ресурсы контекста сразу после ответа

		if err != nil {
			// Если gRPC сервер упал, финансовая система не должна встать колом. 
			// Логируем ошибку и продолжаем бизнес-логику (паттерн Отказоустойчивости)
			log.Printf("[gRPC Error] Could not fetch balance for user %s: %v", event.UserID, err)
			walletData = &walletgrpc.GetBalanceResponse{Balance: 0, Status: "UNKNOWN_ERROR"}
		}

		// Выводим красивое Push-сообщение, обогащенное данными из gRPC
		fmt.Printf(
			"\n[PUSH NOTIFICATION SENT] Юзер '%s': Операция %s на сумму %.2f руб. проведена. Текущий остаток на счете: %.2f руб. (Статус: %s)\n", 
			event.UserID, event.Type, 
			float64(event.Amount)/100.0, 
			float64(walletData.GetBalance())/100.0, 
			walletData.GetStatus())

		// === Записываем транзакцию в Redis (Sorted Set - ZSET) по ключу юзера
		// В качестве Score используем текущее время Unix, чтобы история была отсортирована по хронологии
		redisKey := fmt.Sprintf("history:%s", event.UserID)
		
		err = rdb.ZAdd(ctx, redisKey, redis.Z{
			Score:  float64(time.Now().UnixNano()),
			Member: msg.Value, // Кладем сырой JSON-транзакции как строку
		}).Err()

		if err != nil {
			log.Printf("[Notification Error] Failed to save history to Redis: %v", err)
		} else {
			fmt.Printf("[Redis Cache] Transaction saved to history for user %s\n", event.UserID)
		}
	}

	fmt.Println("[Notification] Worker stopped cleanly. Goodbye!")
}
