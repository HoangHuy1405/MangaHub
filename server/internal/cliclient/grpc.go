package cliclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "mangahub/proto/manga"
)

// GRPCClient manages a gRPC connection to the MangaService.
// Follows the same architecture as TCPClient, UDPClient, WSClient.
type GRPCClient struct {
	cfg    *CLIConfig
	conn   *grpc.ClientConn
	client pb.MangaServiceClient
}

// NewGRPCClient creates a GRPCClient (not yet connected).
func NewGRPCClient(cfg *CLIConfig) *GRPCClient {
	return &GRPCClient{cfg: cfg}
}

// Connect dials the gRPC server with a 5-second timeout.
//
// Skill checklist #5: Client uses context.WithTimeout — never bare context.Background()
// Skill checklist #6: defer conn.Close() is the caller's responsibility via Close()
func (g *GRPCClient) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, g.cfg.GRPCAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("cannot connect to gRPC server at %s — is grpc-server running?", g.cfg.GRPCAddr())
	}

	g.conn = conn
	g.client = pb.NewMangaServiceClient(conn)
	return nil
}

// Close shuts down the gRPC connection.
// Skill checklist #6: defer conn.Close() on client side
func (g *GRPCClient) Close() {
	if g.conn != nil {
		g.conn.Close()
		g.conn = nil
	}
}

// GetManga calls the GetManga RPC with a 5-second timeout.
func (g *GRPCClient) GetManga(mangaID string) (*pb.MangaResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return g.client.GetManga(ctx, &pb.GetMangaRequest{
		MangaId: mangaID,
	})
}

// SearchManga calls the SearchManga RPC with a 5-second timeout.
func (g *GRPCClient) SearchManga(query, genre, status string, page, limit int) (*pb.SearchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return g.client.SearchManga(ctx, &pb.SearchRequest{
		Query:  query,
		Genre:  genre,
		Status: status,
		Page:   int32(page),
		Limit:  int32(limit),
	})
}

// UpdateProgress calls the UpdateProgress RPC with a 5-second timeout.
func (g *GRPCClient) UpdateProgress(userID, mangaID string, chapter int) (*pb.ProgressResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return g.client.UpdateProgress(ctx, &pb.ProgressRequest{
		UserId:    userID,
		MangaId:   mangaID,
		Chapter:   int32(chapter),
		UpdatedAt: time.Now().Unix(),
	})
}
