package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jdashel/slykat-chat-app-proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

type Payload struct {
	Email string
}

type server struct {
	pb.UnimplementedPresenceServiceServer
	db *pgxpool.Pool
}

func (s *server) SetStatus(ctx context.Context, req *pb.SetStatusRequest) (*pb.SetStatusResponse, error) {
	userID, err := strconv.Atoi(req.UserId)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	query := `INSERT INTO presence (user_id, status) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET status = EXCLUDED.status`
	_, err = s.db.Exec(ctx, query, userID, req.Status)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to set status: %v", err)
	}

	return &pb.SetStatusResponse{Success: true}, nil
}

func (s *server) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	userID, err := strconv.Atoi(req.UserId)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	var dbStatus string
	var lastActive time.Time
	err = s.db.QueryRow(ctx, `SELECT status, last_active FROM presence WHERE user_id = $1`, userID).Scan(&dbStatus, &lastActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, grpcstatus.Errorf(codes.NotFound, "user not found")
		}
		return nil, grpcstatus.Errorf(codes.Internal, "failed to get status: %v", err)
	}

	return &pb.GetStatusResponse{
		UserId:     req.UserId,
		Status:     dbStatus,
		LastActive: lastActive.Format(time.RFC3339),
	}, nil
}

func initDatabase(ctx context.Context, db *pgxpool.Pool) error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS presence (
		user_id SERIAL PRIMARY KEY,
		status VARCHAR(20) NOT NULL,
		last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`
	_, err := db.Exec(ctx, createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/slykat-chat-app?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	err = initDatabase(ctx, db)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterPresenceServiceServer(s, &server{db: db})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
