package main

import (
	"log"
	"net"
	"os"

	"billing_service/dao"
	"billing_service/handler"
	"billing_service/pb"
	"billing_service/repository"
	"billing_service/service"

	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "root:@Shinratensei26@tcp(127.0.0.1:3306)/default?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&dao.Loan{}, &dao.LoanSchedule{}, &dao.Payment{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	repo := repository.NewRepository(db)
	svc := service.NewBillingService(repo)
	h := handler.NewHandler(svc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterBillingServiceServer(grpcServer, h)

	log.Printf("gRPC server listening on :%s\n", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
