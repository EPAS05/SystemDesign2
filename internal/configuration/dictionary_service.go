package configuration

import "context"

func validateDictionaryFields(code, name string) error {
	if code == "" || name == "" {
		return ErrInvalidDictionary
	}
	return nil
}

func validateHourlyRate(rate string) error {
	value, ok := parseQuantity(rate)
	if !ok || value < 0 {
		return ErrInvalidDictionary
	}
	return nil
}

func (s *Service) ListOperationClasses(ctx context.Context) ([]*OperationClass, error) {
	return s.repo.ListOperationClasses(ctx)
}

func (s *Service) GetOperationClass(ctx context.Context, id string) (*OperationClass, error) {
	if id == "" {
		return nil, ErrNotFound
	}
	return s.repo.GetOperationClass(ctx, id)
}

func (s *Service) CreateOperationClass(ctx context.Context, code, name, hourlyRate string) (*OperationClass, error) {
	if err := validateDictionaryFields(code, name); err != nil {
		return nil, err
	}
	if err := validateHourlyRate(hourlyRate); err != nil {
		return nil, err
	}
	return s.repo.CreateOperationClass(ctx, code, name, hourlyRate)
}

func (s *Service) UpdateOperationClass(ctx context.Context, id, code, name, hourlyRate string) (*OperationClass, error) {
	if id == "" {
		return nil, ErrNotFound
	}
	if err := validateDictionaryFields(code, name); err != nil {
		return nil, err
	}
	if err := validateHourlyRate(hourlyRate); err != nil {
		return nil, err
	}
	return s.repo.UpdateOperationClass(ctx, id, code, name, hourlyRate)
}

func (s *Service) DeleteOperationClass(ctx context.Context, id string) error {
	if id == "" {
		return ErrNotFound
	}
	return s.repo.DeleteOperationClass(ctx, id)
}

func (s *Service) ListEquipment(ctx context.Context) ([]*Equipment, error) {
	return s.repo.ListEquipment(ctx)
}

func (s *Service) GetEquipment(ctx context.Context, id string) (*Equipment, error) {
	if id == "" {
		return nil, ErrNotFound
	}
	return s.repo.GetEquipment(ctx, id)
}

func (s *Service) CreateEquipment(ctx context.Context, code, name string) (*Equipment, error) {
	if err := validateDictionaryFields(code, name); err != nil {
		return nil, err
	}
	return s.repo.CreateEquipment(ctx, code, name)
}

func (s *Service) UpdateEquipment(ctx context.Context, id, code, name string) (*Equipment, error) {
	if id == "" {
		return nil, ErrNotFound
	}
	if err := validateDictionaryFields(code, name); err != nil {
		return nil, err
	}
	return s.repo.UpdateEquipment(ctx, id, code, name)
}

func (s *Service) DeleteEquipment(ctx context.Context, id string) error {
	if id == "" {
		return ErrNotFound
	}
	return s.repo.DeleteEquipment(ctx, id)
}

func (s *Service) ListTools(ctx context.Context) ([]*Tool, error) {
	return s.repo.ListTools(ctx)
}

func (s *Service) GetTool(ctx context.Context, id string) (*Tool, error) {
	if id == "" {
		return nil, ErrNotFound
	}
	return s.repo.GetTool(ctx, id)
}

func (s *Service) CreateTool(ctx context.Context, code, name string) (*Tool, error) {
	if err := validateDictionaryFields(code, name); err != nil {
		return nil, err
	}
	return s.repo.CreateTool(ctx, code, name)
}

func (s *Service) UpdateTool(ctx context.Context, id, code, name string) (*Tool, error) {
	if id == "" {
		return nil, ErrNotFound
	}
	if err := validateDictionaryFields(code, name); err != nil {
		return nil, err
	}
	return s.repo.UpdateTool(ctx, id, code, name)
}

func (s *Service) DeleteTool(ctx context.Context, id string) error {
	if id == "" {
		return ErrNotFound
	}
	return s.repo.DeleteTool(ctx, id)
}
