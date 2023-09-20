package store

import (
	"context"

	"github.com/metal-toolbox/disko/internal/model"
)

type Repository interface {
	// AssetByID returns asset.
	AssetByID(ctx context.Context, id string) (*model.Asset, error)
}
