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
	pb.UnimplementedChatServiceServer
	db *pgxpool.Pool
}

func (s *server) CreateChannel(ctx context.Context, req *pb.CreateChannelRequest) (*pb.CreateChannelResponse, error) {
	var channelID int32
	err := s.db.QueryRow(ctx, "INSERT INTO channels (name, creator_id) VALUES ($1, $2) RETURNING id", req.Name, req.CreatorId).Scan(&channelID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create channel: %v", err)
	}
	return &pb.CreateChannelResponse{ChannelId: fmt.Sprintf("%d", channelID), Name: req.Name}, nil
}

func (s *server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	var messageID int32
	err := s.db.QueryRow(ctx, "INSERT INTO messages (channel_id, sender_id, content) VALUES ($1, $2, $3) RETURNING id", req.ChannelId, req.SenderId, req.Content).Scan(&messageID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send message: %v", err)
	}
	return &pb.SendMessageResponse{MessageId: fmt.Sprintf("%d", messageID)}, nil
}

func (s *server) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	rows, err := s.db.Query(ctx, "SELECT id, channel_id, sender_id, content, created_at FROM messages WHERE channel_id = $1 LIMIT $2 OFFSET $3", req.ChannelId, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get messages: %v", err)
	}
	defer rows.Close()

	var messages []*pb.Message
	for rows.Next() {
		var message pb.Message
		err := rows.Scan(&message.Id, &message.ChannelId, &message.SenderId, &message.Content, &message.CreatedAt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan row: %v", err)
		}
		messages = append(messages, &message)
	}

	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "rows error: %v", err)
	}

	return &pb.GetMessagesResponse{Messages: messages}, nil
}

func initDatabase(ctx context.Context, db *pgxpool.Pool) error {
	createChannelsTable := `
	CREATE TABLE IF NOT EXISTS channels (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		creator_id VARCHAR(255) NOT NULL
	);
	`
	createMessagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id SERIAL PRIMARY KEY,
		channel_id INTEGER NOT NULL,
		sender_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.Exec(ctx, createChannelsTable)
	if err != nil {
		return fmt.Errorf("failed to create channels table: %w", err)
	}

	_, err = db.Exec(ctx, createMessagesTable)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
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

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, &server{db: db})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
