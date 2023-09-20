package store

import (
	"context"

	"github.com/metal-toolbox/disko/internal/model"
)

type Mock struct{}

func NewMockInventory() (Repository, error) {
	return &Mock{}, nil
}

// AssetByID returns device attributes by its identifier
func (s *Mock) AssetByID(_ context.Context, _ string) (*model.Asset, error) {
	return nil, nil
}
