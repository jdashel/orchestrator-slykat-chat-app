package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jdashel/slykat-chat-app-proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Payload struct {
	Email string
}

type server struct {
	pb.UnimplementedAuthServiceServer
	db *pgxpool.Pool
}

func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	var userID int32
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`, req.Username, req.Email, req.Password).Scan(&userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}
	token := fmt.Sprintf("token_%d", userID)
	_, err = s.db.Exec(ctx, `
		UPDATE users
		SET token = $1
		WHERE id = $2
	`, token, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user token: %v", err)
	}
	return &pb.RegisterResponse{
		UserId: fmt.Sprintf("%d", userID),
		Token:  token,
	}, nil
}

func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	var userID int32
	err := s.db.QueryRow(ctx, `
		SELECT id FROM users WHERE email = $1 AND password_hash = $2
	`, req.Email, req.Password).Scan(&userID)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}
	token := fmt.Sprintf("token_%d", userID)
	_, err = s.db.Exec(ctx, `
		UPDATE users
		SET token = $1
		WHERE id = $2
	`, token, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user token: %v", err)
	}
	return &pb.LoginResponse{
		Token:  token,
		UserId: fmt.Sprintf("%d", userID),
	}, nil
}

func (s *server) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	var userID int32
	err := s.db.QueryRow(ctx, `
		SELECT id FROM users WHERE token = $1
	`, req.Token).Scan(&userID)
	if err != nil {
		return &pb.ValidateTokenResponse{
			UserId:  "",
			IsValid: false,
		}, nil
	}
	return &pb.ValidateTokenResponse{
		UserId:  fmt.Sprintf("%d", userID),
		IsValid: true,
	}, nil
}

func initDatabase(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			token VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}
	return nil
}

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/slykat-chat-app?sslmode=disable")
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer db.Close()

	err = initDatabase(ctx, db)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v\n", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &server{db: db})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
