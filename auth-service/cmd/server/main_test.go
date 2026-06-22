package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jdashel/slykat-chat-app-proto"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/slykat-chat-app?sslmode=disable")
	if err != nil {
		t.Skip("Failed to connect to database:", err)
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		t.Skip("Database connection failed:", err)
	}

	initDatabase(ctx, db)

	s := &server{db: db}

	username := fmt.Sprintf("testuser%d", time.Now().UnixNano())
	email := username + "@example.com"
	password := "password123"

	req := &pb.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	res, err := s.Register(ctx, req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if res.UserId == "" || res.Token == "" {
		t.Errorf("Invalid response: user_id=%s, token=%s", res.UserId, res.Token)
	}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/slykat-chat-app?sslmode=disable")
	if err != nil {
		t.Skip("Failed to connect to database:", err)
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		t.Skip("Database connection failed:", err)
	}

	initDatabase(ctx, db)

	s := &server{db: db}

	username := fmt.Sprintf("testuser%d", time.Now().UnixNano())
	email := username + "@example.com"
	password := "password123"

	reqRegister := &pb.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	resRegister, err := s.Register(ctx, reqRegister)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if resRegister.UserId == "" || resRegister.Token == "" {
		t.Errorf("Invalid response: user_id=%s, token=%s", resRegister.UserId, resRegister.Token)
	}

	reqLogin := &pb.LoginRequest{
		Email:    email,
		Password: password,
	}

	resLogin, err := s.Login(ctx, reqLogin)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if resLogin.Token == "" || resLogin.UserId != resRegister.UserId {
		t.Errorf("Invalid response: token=%s, user_id=%s", resLogin.Token, resLogin.UserId)
	}
}
