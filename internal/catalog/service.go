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

func (s *Service) CreateComponent(ctx context.Context, nodeID, name string) (*Component, error) {
	if _, err := s.repo.Get(ctx, nodeID); err != nil {
		return nil, err
	}
	return s.repo.CreateComponent(ctx, nodeID, name)
}

func (s *Service) GetComponent(ctx context.Context, id string) (*Component, error) {
	return s.repo.GetComponent(ctx, id)
}

func (s *Service) ListComponentsByNode(ctx context.Context, nodeID string) ([]*Component, error) {
	return s.repo.ListComponentsByNode(ctx, nodeID)
}

func (s *Service) UpdateComponent(ctx context.Context, id, name string) (*Component, error) {
	return s.repo.UpdateComponent(ctx, id, name)
}

func (s *Service) DeleteComponent(ctx context.Context, id string) error {
	return s.repo.DeleteComponent(ctx, id)
}

func (s *Service) CreateProduct(ctx context.Context, nodeID, name string) (*Product, error) {
	if _, err := s.repo.Get(ctx, nodeID); err != nil {
		return nil, err
	}
	return s.repo.CreateProduct(ctx, nodeID, name)
}

func (s *Service) GetProduct(ctx context.Context, id string) (*Product, error) {
	return s.repo.GetProduct(ctx, id)
}

func (s *Service) ListProductsByNode(ctx context.Context, nodeID string) ([]*Product, error) {
	return s.repo.ListProductsByNode(ctx, nodeID)
}

func (s *Service) UpdateProduct(ctx context.Context, id, name string) (*Product, error) {
	return s.repo.UpdateProduct(ctx, id, name)
}

func (s *Service) DeleteProduct(ctx context.Context, id string) error {
	return s.repo.DeleteProduct(ctx, id)
}
