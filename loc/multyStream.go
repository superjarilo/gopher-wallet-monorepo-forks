package main

/**
 * 	Задача https://go.dev/play/p/CwS3W4HiE_S:
 *		Реализовать функцию асинхронного сбора данных из внешнего источника (например, запрос к API или тяжелый расчет в БД)
 *		с жестким таймаутом выполнения. Если источник не ответил за 500мс — вернуть дефолтное значение или ошибку,
 *		не заставляя клиента ждать и предотвращая зависание горутины.
 */

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Имитируем тяжелый запрос к БД или стороннему API
func fetchUserData(ctx context.Context, userID string) (string, error) {
	// В реальной жизни тут будет HTTP-запрос или SQL-запрос
	time.Sleep(600 * time.Millisecond) // Имитируем долгий ответ (дольше нашего таймаута)
	return "User Data: Maria, Senior Go Developer", nil
}

// Решение задачи CwS3W4HiE_S через оператор select
func FetchDataWithTimeout(userID string, timeout time.Duration) (string, error) {
	// Создаем контекст с таймаутом. Он автоматически закроет канал ctx.Done(), когда время выйдет
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // Всегда освобождаем ресурсы контекста!

	// Создаем буферизированный канал на 1 элемент.
	// ВАЖНО: Если канал сделать НЕбуферизированным, и функция выйдет по таймауту,
	// горутина ниже заблокируется на отправке навсегда (утечка памяти / Goroutine Leak).
	ch := make(chan string, 1)
	errCh := make(chan error, 1)

	// Запускаем асинхронную горутину для выполнения тяжелой задачи
	go func() {
		data, err := fetchUserData(ctx, userID)
		if err != nil {
			errCh <- err
			return
		}
		ch <- data
	}()

	// Оператор select блокирует выполнение и ждет, какой из каналов выстрелит первым
	select {
	case result := <-ch:
		// Данные успешно получены раньше, чем сработал таймаут
		return result, nil

	case err := <-errCh:
		// Получена ошибка в процессе выполнения задачи
		return "", err

	case <-ctx.Done():
		// Сработал таймаут! Канал ctx.Done() закрылся.
		// Мы мгновенно возвращаем управление пользователю,
		// не дожидаясь завершения fetchUserData
		return "", errors.New("request timed out: service responded too slow")
	}
}

func main() {
	// Вызываем функцию с таймаутом в 500 миллисекунд
	result, err := FetchDataWithTimeout("42", 500*time.Millisecond)
	if err != nil {
		// Ожидаем ошибку таймаута, так как имитация длится 600мс
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	fmt.Printf("[SUCCESS] %s\n", result)
}
