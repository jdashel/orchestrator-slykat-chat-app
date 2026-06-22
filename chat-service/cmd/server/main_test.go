package main

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jdashel/slykat-chat-app-proto"
)

func TestCreateChannel(t *testing.T) {
	connStr := "postgres://postgres:postgres@localhost:5432/slykat-chat-app?sslmode=disable"
	db, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Skip("Failed to connect to database:", err)
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		t.Skip("Database is not reachable:", err)
	}

	initDatabase(context.Background(), db)

	s := &server{db: db}

	req := &pb.CreateChannelRequest{
		Name:      fmt.Sprintf("test_channel_%d", time.Now().UnixNano()),
		CreatorId: fmt.Sprintf("%d", rand.Intn(1000000)),
	}
	resp, err := s.CreateChannel(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	if resp.ChannelId == "" || resp.Name != req.Name {
		t.Errorf("Invalid response: %+v", resp)
	}
}

func TestSendMessage(t *testing.T) {
	connStr := "postgres://postgres:postgres@localhost:5432/slykat-chat-app?sslmode=disable"
	db, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Skip("Failed to connect to database:", err)
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		t.Skip("Database is not reachable:", err)
	}

	initDatabase(context.Background(), db)

	s := &server{db: db}

	createReq := &pb.CreateChannelRequest{
		Name:      fmt.Sprintf("test_channel_%d", time.Now().UnixNano()),
		CreatorId: fmt.Sprintf("%d", rand.Intn(1000000)),
	}
	createResp, err := s.CreateChannel(context.Background(), createReq)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	sendReq := &pb.SendMessageRequest{
		ChannelId: createResp.ChannelId,
		SenderId:  fmt.Sprintf("%d", rand.Intn(1000000)),
		Content:   "Hello, world!",
	}
	resp, err := s.SendMessage(context.Background(), sendReq)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if resp.MessageId == "" || resp.CreatedAt == "" {
		t.Errorf("Invalid response: %+v", resp)
	}
}
