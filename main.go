package main

import (
	"context"
	"flag"
	"hm-dianping-go/config"
	"hm-dianping-go/dao"
	"hm-dianping-go/models"
	"hm-dianping-go/router"
	"hm-dianping-go/service"
	"hm-dianping-go/utils"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log"
)

func main() {
	configPath := flag.String("config", "config/application.yaml", "Path to configuration file")
	flag.Parse()
	if err := config.LoadConfigFromFile(*configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if err := dao.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	if err := dao.InitRedis(); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	if err := dao.DB.AutoMigrate(
		&models.User{},
		&models.Shop{},
		&models.ShopType{},
		&models.Voucher{},
		&models.VoucherOrder{},
		&models.Blog{},
		&models.Follow{},
		&models.BlogLike{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	if err := initBloomFilters(); err != nil {
		log.Printf("Warning: Failed to initialize bloom filters: %v", err)
	}
	if err := service.InitKafkaConsumer(); err != nil {
		log.Fatalf("Failed to initialize Kafka consumer: %v", err)
	}
	r := router.SetupRouter()
	port := config.GetConfig().Server.Port
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	// 在goroutine中启动服务器
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 停止Kafka消费者
	service.StopKafkaConsumers()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
func initBloomFilters() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// 使用函数适配器创建IDProvider，解耦数据库依赖
	provider := utils.IDProviderFunc{
		ShopIDsFunc:    dao.GetAllShopIDsWithContext,
		UserIDsFunc:    dao.GetAllUserIDsWithContext,
		VoucherIDsFunc: dao.GetAllVoucherIDsWithContext,
	}
	initializer := utils.NewBloomInitializer(dao.Redis, provider)
	if err := initializer.InitAllBloomFilters(ctx); err != nil {
		return err
	}
	health := initializer.CheckBloomFilterHealth(ctx)
	log.Printf("布隆过滤器健康状态: %+v", health)

	return nil
}
