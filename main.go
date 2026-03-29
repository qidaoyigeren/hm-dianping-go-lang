package main

import (
	"flag"
	"hm-dianping-go/config"
	"hm-dianping-go/dao"
	"hm-dianping-go/models"
	"hm-dianping-go/router"

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
	r := router.SetupRouter()
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
