package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/Albitko/loyalty-program/internal/config"
	"github.com/Albitko/loyalty-program/internal/controller"
	"github.com/Albitko/loyalty-program/internal/repo"
	"github.com/Albitko/loyalty-program/internal/usecase"
	"github.com/Albitko/loyalty-program/internal/utils"
	"github.com/Albitko/loyalty-program/internal/workers"
)

func init() {
	utils.InitializeLogger()
}

func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() { _ = utils.Logger.Sync() }()
	cfg, err := config.New()
	if err != nil {
		panic(fmt.Errorf("create config failed: %w", err))
	}

	utils.Logger.Info("program started")
	defer utils.Logger.Info("program finished")

	storage := repo.NewRepository(ctx, cfg.DatabaseURI)
	defer storage.Close()

	queue := workers.InitWorkers(ctx, storage, "https://test-service.com")

	userAuthenticator := usecase.NewAuthenticator(storage)
	ordersProcessor := usecase.NewOrdersProcessor(storage, queue)
	balanceProcessor := usecase.NewBalanceProcessor(storage)

	userHandler := controller.NewUserAuthHandler(userAuthenticator)
	ordersHandler := controller.NewOrdersHandler(ordersProcessor)
	balanceHandler := controller.NewBalanceHandler(balanceProcessor)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.POST("/api/user/register", userHandler.Register)
	r.POST("/api/user/login", userHandler.Login)

	authorized := r.Group("/api/user/")
	// authorized.Use() JWT middleware
	authorized.POST("orders", ordersHandler.CreateOrder)
	authorized.GET("orders", ordersHandler.GetOrders)
	authorized.GET("balance", balanceHandler.GetBalance)
	authorized.GET("balance/withdraw", balanceHandler.Withdraw)
	authorized.GET("withdrawals", balanceHandler.GetWithdrawn)

	err = r.Run(cfg.RunAddress)
	if err != nil {
		panic(fmt.Errorf("start server failed: %w", err))
	}
}
