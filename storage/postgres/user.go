package postgres

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
	users "test/genproto/user"
)

type UserRepository struct {
	Db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{Db: db}
}

func (r *UserRepository) Register(ctx context.Context, user *users.RegisterRequest) (string, string, error) {
	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return "", "", err
	}

	var userID, role string
	query := `INSERT INTO users (email, password_hash, first_name, last_name) VALUES ($1, $2, $3, $4) RETURNING id, role`
	err = r.Db.QueryRowContext(ctx, query, user.Email, hashedPassword, user.FirstName, user.LastName).Scan(&userID, &role)
	if err != nil {
		return "", "", err
	}

	// Generate token (mocked for now)
	return userID, role, nil
}

func (r *UserRepository) Login(ctx context.Context, req *users.LoginRequest) (string, string, error) {
	var storedPassword, id, role string
	query := `SELECT password_hash, id, role FROM users WHERE email = $1`
	err := r.Db.QueryRowContext(ctx, query, req.Email).Scan(&storedPassword, &id, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", errors.New("user not found")
		}
		return "", "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(req.Password))
	if err != nil {
		return "", "", errors.New("invalid password")
	}

	// Generate token (mocked for now)
	return id, role, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, req *users.UpdateUserRequest, id string) error {
	query := `UPDATE users SET first_name = $1, last_name = $2 WHERE id = $3`
	_, err := r.Db.ExecContext(ctx, query, req.FirstName, req.LastName, id)
	return err
}

func (r *UserRepository) GetUser(ctx context.Context, userId string) (*users.User, error) {
	var user users.User
	query := `SELECT id, email, first_name, last_name, created_at FROM users WHERE id = $1`
	err := r.Db.QueryRowContext(ctx, query, userId).Scan(&user.Id, &user.Email, &user.FirstName, &user.LastName, &user.CreatedAt)
	return &user, err
}

func (r *UserRepository) CreateTodo(ctx context.Context, userId string, req *users.CreateTodoRequest) (*users.Todo, error) {
	var todo users.Todo
	query := `INSERT INTO todos (user_id, title, completed, created_at) VALUES ($1, $2, FALSE, NOW()) RETURNING id, user_id, title, completed, created_at`
	err := r.Db.QueryRowContext(ctx, query, userId, req.Title).Scan(&todo.Id, &todo.UserId, &todo.Title, &todo.Completed, &todo.CreatedAt)
	return &todo, err
}

func (r *UserRepository) GetTodos(ctx context.Context, userId string) (*users.GetTodosResponse, error) {
	query := `SELECT id, user_id, title, completed, created_at FROM todos WHERE user_id = $1`
	rows, err := r.Db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*users.Todo
	for rows.Next() {
		var todo users.Todo
		if err := rows.Scan(&todo.Id, &todo.UserId, &todo.Title, &todo.Completed, &todo.CreatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, &todo)
	}
	return &users.GetTodosResponse{Todos: todos}, nil
}

func (r *UserRepository) UpdateTodo(ctx context.Context, req *users.UpdateTodoRequest) (*users.Todo, error) {
	query := `UPDATE todos SET title = $1, completed = $2 WHERE id = $3 RETURNING id, user_id, title, completed, created_at`
	var todo users.Todo
	err := r.Db.QueryRowContext(ctx, query, req.Title, req.Completed, req.Id).Scan(&todo.Id, &todo.UserId, &todo.Title, &todo.Completed, &todo.CreatedAt)
	return &todo, err
}

func (r *UserRepository) DeleteTodo(ctx context.Context, req *users.DeleteRequest) error {
	query := `DELETE FROM todos WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, req.Id)
	return err
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("failed to hash password")
	}
	return string(hashed), nil
}
