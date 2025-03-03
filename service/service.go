package service

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

	"test/auth"
	pb "test/genproto/user"
	"test/storage/postgres"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type UserService struct {
	pb.UnimplementedUserServiceServer
	logger *slog.Logger
	store  *postgres.UserRepository
}

func NewUserService(db *sql.DB, logger *slog.Logger) *UserService {
	return &UserService{
		store:  postgres.NewUserRepository(db),
		logger: logger,
	}
}

func (s *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.TokenResponse, error) {
	id, role, err := s.store.Register(ctx, req)
	if err != nil {
		s.logger.Error("failed to register user", slog.String("error", err.Error()))
		return nil, err
	}
	token, err := auth.GenerateJWTToken(id, role)
	return &pb.TokenResponse{AccessToken: token}, nil
}

func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.TokenResponse, error) {
	id, role, err := s.store.Login(ctx, req)
	if err != nil {
		s.logger.Error("failed to login user", slog.String("error", err.Error()))
		return nil, err
	}
	token, err := auth.GenerateJWTToken(id, role)
	return &pb.TokenResponse{AccessToken: token}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.Empty, error) {
	userId, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	err = s.store.UpdateUser(ctx, req, userId)
	if err != nil {
		s.logger.Error("failed to update user", slog.String("error", err.Error()))
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *pb.Empty) (*pb.User, error) {
	userId, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	user, err := s.store.GetUser(ctx, userId)
	if err != nil {
		s.logger.Error("failed to get user", slog.String("error", err.Error()))
		return nil, err
	}
	return user, nil
}

func (s *UserService) CreateTodo(ctx context.Context, req *pb.CreateTodoRequest) (*pb.Todo, error) {
	userId, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	todo, err := s.store.CreateTodo(ctx, userId, req)
	if err != nil {
		s.logger.Error("failed to create todo", slog.String("error", err.Error()))
		return nil, err
	}
	return todo, nil
}

func (s *UserService) GetTodos(ctx context.Context, req *pb.Empty) (*pb.GetTodosResponse, error) {
	userId, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	todos, err := s.store.GetTodos(ctx, userId)
	if err != nil {
		s.logger.Error("failed to get todos", slog.String("error", err.Error()))
		return nil, err
	}
	return todos, nil
}

func (s *UserService) UpdateTodo(ctx context.Context, req *pb.UpdateTodoRequest) (*pb.Todo, error) {
	todo, err := s.store.UpdateTodo(ctx, req)
	if err != nil {
		s.logger.Error("failed to update todo", slog.String("error", err.Error()))
		return nil, err
	}
	return todo, nil
}

func (s *UserService) DeleteTodo(ctx context.Context, req *pb.DeleteRequest) (*pb.Empty, error) {
	err := s.store.DeleteTodo(ctx, req)
	if err != nil {
		s.logger.Error("failed to delete todo", slog.String("error", err.Error()))
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *UserService) getUserIDFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		s.logger.Error("metadata not found in context")
		return "", status.Error(codes.Unauthenticated, "authentication required")
	}

	authHeader := md.Get("Authorization")
	if len(authHeader) == 0 {
		s.logger.Error("Authorization header missing")
		return "", status.Error(codes.Unauthenticated, "Authorization token required")
	}

	token := strings.TrimPrefix(authHeader[0], "Bearer ")
	userID, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		s.logger.Error("invalid token", "error", err)
		return "", status.Error(codes.Unauthenticated, "invalid token")
	}

	return userID, nil
}
