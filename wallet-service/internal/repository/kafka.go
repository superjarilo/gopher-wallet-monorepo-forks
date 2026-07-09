package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// TransactionEvent описывает структуру события для брокера сообщений
type TransactionEvent struct {
	UserID string `json:"user_id"`
	Amount int64  `json:"amount"` // Списание передается как положительное число, тип регулируется полем Type
	Type   string `json:"type"`   // 'DEPOSIT' или 'DEBIT'
}

type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher инициализирует продюсер для работы с очередями
func NewKafkaPublisher(broker string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(broker),
			Topic:    "transactions",
			Balancer: &kafka.LeastBytes{}, // Равномерно распределяет нагрузку по партициям
		},
	}
}

// PublishTransfer отправляет событие о движении средств в Kafka
func (p *KafkaPublisher) PublishTransfer(ctx context.Context, userID string, amount int64, txType string) error {
	event := TransactionEvent{
		UserID: userID,
		Amount: amount,
		Type:   txType,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka event serialization failed: %w", err)
	}

	// Записываем бинарный JSON в топик брокера
	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(userID), // Ключ по userID гарантирует строгую последовательность операций одного юзера
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("kafka write message failed: %w", err)
	}

	return nil
}

// Close корректно закрывает соединение при Graceful Shutdown приложения
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
