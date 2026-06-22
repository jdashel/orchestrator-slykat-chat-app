package main

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jdashel/slykat-chat-app-proto"
)

func TestSetStatus(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/slykat-chat-app?sslmode=disable")
	if err != nil {
		t.Skip("Failed to connect to database:", err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		t.Skip("Database is not reachable:", err)
	}

	initDatabase(ctx, db)

	s := &server{db: db}

	userID := fmt.Sprintf("%d", rand.Intn(1000000))
	status := "online"

	req := &pb.SetStatusRequest{
		UserId: userID,
		Status: status,
	}
	resp, err := s.SetStatus(ctx, req)
	if err != nil {
		t.Fatalf("Failed to set status: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success to be true, got false")
	}
}

func TestGetStatus(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/slykat-chat-app?sslmode=disable")
	if err != nil {
		t.Skip("Failed to connect to database:", err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		t.Skip("Database is not reachable:", err)
	}

	initDatabase(ctx, db)

	s := &server{db: db}

	userID := fmt.Sprintf("%d", rand.Intn(1000000))
	status := "online"

	setReq := &pb.SetStatusRequest{
		UserId: userID,
		Status: status,
	}
	_, err = s.SetStatus(ctx, setReq)
	if err != nil {
		t.Fatalf("Failed to set status: %v", err)
	}

	getReq := &pb.GetStatusRequest{
		UserId: userID,
	}
	resp, err := s.GetStatus(ctx, getReq)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if resp.UserId != userID {
		t.Errorf("Expected user_id to be %s, got %s", userID, resp.UserId)
	}
	if resp.Status != status {
		t.Errorf("Expected status to be %s, got %s", status, resp.Status)
	}
}
