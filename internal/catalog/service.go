package catalog

import "context"

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateNode(ctx context.Context, parentID *string, name string) (*Node, error) {
	return s.repo.Create(ctx, parentID, name)
}

func (s *Service) GetNode(ctx context.Context, id string) (*Node, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) ListChildren(ctx context.Context, parentID string) ([]*Node, error) {
	return s.repo.ListChildren(ctx, parentID)
}

func (s *Service) UpdateNode(ctx context.Context, id, name string) (*Node, error) {
	return s.repo.Update(ctx, id, name)
}

func (s *Service) DeleteNode(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
