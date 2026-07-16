package grpc

import (
	"context"
	"errors"

	// Импортируем сгенерированный код контракта
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/pkg/walletgrpc"
)

// Описываем, что сервер хочет от бизнес-логики (Use Case)
// (Нам нужно будет добавить метод GetWallet в usecase)
type WalletUseCase interface {
	GetWallet(ctx context.Context, userID string) (walletID string, balance int64, status string, currency string, err error)
}

type Server struct {
	walletgrpc.UnimplementedWalletServiceServer // Обязательная заглушка для обратной совместимости
	uc WalletUseCase
}

func NewServer(uc WalletUseCase) *Server {
	return &Server{uc: uc}
}

// GetBalance реализует gRPC-метод из нашего .proto файла
func (s *Server) GetBalance(ctx context.Context, req *walletgrpc.GetBalanceRequest) (*walletgrpc.GetBalanceResponse, error) {
	if req.GetUserId() == "" {
		return nil, errors.New("user_id is required")
	}

	walletID, balance, status, currency, err := s.uc.GetWallet(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}

	// Возвращаем сгенерированную Protobuf структуру
	return &walletgrpc.GetBalanceResponse{
		WalletId: walletID,
		Balance:  balance,
		Status:   status,
		Currency: currency,
	}, nil
}
