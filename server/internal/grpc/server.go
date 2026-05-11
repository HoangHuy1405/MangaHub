package grpc

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
	pb "mangahub/proto/manga"
)

// mangaServer implements the MangaServiceServer gRPC interface.
//
// Design: reuses the existing Repository layer (DRY) — no duplicate DB logic.
// Follows the grpc-protobuf-sqlite skill pattern:
//   - Embed UnimplementedMangaServiceServer for forward compatibility (checklist #1)
//   - All errors via status.Error(codes.X, msg) — never fmt.Errorf (checklist #3)
//   - Input validated before DB call → codes.InvalidArgument early (checklist #7)
type mangaServer struct {
	pb.UnimplementedMangaServiceServer // Skill checklist #1: MUST embed for forward compat
	mangaRepo repository.MangaRepository
	userRepo  repository.UserRepository
}

// NewMangaServer creates a gRPC server backed by the given repositories.
func NewMangaServer(mangaRepo repository.MangaRepository, userRepo repository.UserRepository) pb.MangaServiceServer {
	return &mangaServer{
		mangaRepo: mangaRepo,
		userRepo:  userRepo,
	}
}

// GetManga retrieves a single manga by ID.
//
// Skill compliance:
//   - Input validation → codes.InvalidArgument (checklist #7)
//   - nil result → codes.NotFound (checklist #4)
//   - DB error → codes.Internal (checklist #4)
func (s *mangaServer) GetManga(ctx context.Context, req *pb.GetMangaRequest) (*pb.MangaResponse, error) {
	log.Printf("[gRPC] GetManga request: manga_id=%s", req.GetMangaId())

	// Validate input (skill checklist #7).
	if req.GetMangaId() == "" {
		return nil, status.Error(codes.InvalidArgument, "manga_id is required")
	}

	manga, err := s.mangaRepo.FindByID(req.GetMangaId())
	if err != nil {
		log.Printf("[gRPC] GetManga DB error: %v", err)
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}
	if manga == nil {
		return nil, status.Errorf(codes.NotFound, "manga not found: id=%s", req.GetMangaId())
	}

	log.Printf("[gRPC] GetManga found: id=%s title=%s", manga.ID, manga.Title)
	return mangaToProto(manga), nil
}

// SearchManga performs a filtered search across the manga catalog.
//
// Skill compliance:
//   - Uses page/limit defaults for safety
//   - Maps results to repeated MangaResponse
func (s *mangaServer) SearchManga(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	log.Printf("[gRPC] SearchManga request: query=%q genre=%q status=%q page=%d limit=%d",
		req.GetQuery(), req.GetGenre(), req.GetStatus(), req.GetPage(), req.GetLimit())

	page := int(req.GetPage())
	limit := int(req.GetLimit())
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	mangaList, total, err := s.mangaRepo.GetAll(req.GetGenre(), req.GetStatus(), req.GetQuery(), "", 0, 0, 0, "", "", page, limit)
	if err != nil {
		log.Printf("[gRPC] SearchManga DB error: %v", err)
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	results := make([]*pb.MangaListItem, 0, len(mangaList))
	for i := range mangaList {
		results = append(results, mangaToListProto(&mangaList[i]))
	}

	log.Printf("[gRPC] SearchManga returning %d results (total=%d)", len(results), total)
	return &pb.SearchResponse{
		Results: results,
		Total:   int32(total),
	}, nil
}

// UpdateProgress updates a user's reading progress for a manga.
//
// Skill compliance:
//   - Input validated before DB call (checklist #7)
//   - rowsAffected == 0 → codes.NotFound (checklist #4)
//   - DB error → codes.Internal (checklist #4)
func (s *mangaServer) UpdateProgress(ctx context.Context, req *pb.ProgressRequest) (*pb.ProgressResponse, error) {
	log.Printf("[gRPC] UpdateProgress request: user_id=%s manga_id=%s chapter=%d",
		req.GetUserId(), req.GetMangaId(), req.GetChapter())

	// Validate input (skill checklist #7).
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.GetMangaId() == "" {
		return nil, status.Error(codes.InvalidArgument, "manga_id is required")
	}
	if req.GetChapter() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "chapter must be > 0")
	}

	// Parse user_id string to int (matches existing repository interface).
	userID, err := strconv.Atoi(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %s (must be numeric)", req.GetUserId())
	}

	// Verify manga exists first.
	exists, err := s.mangaRepo.Exists(req.GetMangaId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check manga existence: %v", err)
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "manga not found: id=%s", req.GetMangaId())
	}

	rowsAffected, err := s.userRepo.UpdateProgress(userID, req.GetMangaId(), int(req.GetChapter()))
	if err != nil {
		log.Printf("[gRPC] UpdateProgress DB error: %v", err)
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}
	if rowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound,
			"manga not found in user's library: user_id=%s, manga_id=%s. Add it first via POST /api/v1/users/library",
			req.GetUserId(), req.GetMangaId())
	}

	msg := fmt.Sprintf("Progress updated: user=%s, manga=%s, chapter=%d",
		req.GetUserId(), req.GetMangaId(), req.GetChapter())
	log.Printf("[gRPC] %s", msg)

	return &pb.ProgressResponse{
		Success: true,
		Message: msg,
	}, nil
}

// mangaToProto maps a Go model to a protobuf MangaResponse.
func mangaToProto(m *models.Manga) *pb.MangaResponse {
	return &pb.MangaResponse{
		Id:            m.ID,
		Title:         m.Title,
		Author:        m.Author,
		Description:   m.Description,
		CoverUrl:      m.CoverURL,
		TotalChapters: int32(m.TotalChapters),
		Status:        m.Status,
		Genres:        m.Genres,
	}
}

// mangaToListProto maps a Go model to a protobuf MangaListItem.
func mangaToListProto(m *models.Manga) *pb.MangaListItem {
	return &pb.MangaListItem{
		Id:            m.ID,
		Title:         m.Title,
		Author:        m.Author,
		CoverUrl:      m.CoverURL,
		TotalChapters: int32(m.TotalChapters),
		Status:        m.Status,
	}
}
