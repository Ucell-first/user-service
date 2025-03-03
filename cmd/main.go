package main

import (
	"context"
	"log"
	"net/http"

	"test/config"
	pb "test/genproto/user"
	"test/logs"
	"test/middleware"

	"test/service"
	"test/storage/postgres"

	"github.com/casbin/casbin/v2"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	conf := config.Load()

	store, err := postgres.ConnectionDb()
	if err != nil {
		log.Fatal(err)
	}

	svc := service.NewUserService(store, logs.NewLogger())

	// Create gRPC server
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	pb.RegisterUserServiceServer(grpcServer, svc)

	// Create HTTP server with gRPC-Gateway
	mux := runtime.NewServeMux()
	err = pb.RegisterUserServiceHandlerServer(context.Background(), mux, svc)
	if err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	// Serve Swagger UI
	fs := http.FileServer(http.Dir("swagger"))
	http.Handle("/swagger/", http.StripPrefix("/swagger/", fs))
	http.Handle("/", mux)

	// Initialize Casbin enforcer
	enforcer, err := casbin.NewEnforcer("casbin/model.conf", "casbin/policy.csv")
	if err != nil {
		log.Fatal(err)
	}

	// Use middleware
	handler := cors.Default().Handler(
		middleware.AuthMiddleware(enforcer)(
			http.DefaultServeMux,
		),
	)

	log.Println("Server running on")
	log.Printf("Swagger UI available at http://localhost%s/swagger/", conf.Server.USER_ROUTER)
	http.ListenAndServe(conf.Server.USER_ROUTER, handler)
}
