package store

import (
	"context"
	"errors"

	"github.com/metal-toolbox/disko/internal/model"
)

const (
	InventorySourceYAML = "inventoryStoreYAML"
)

var (
	ErrYamlSource = errors.New("error in Yaml inventory")
)

// Yaml type implements the inventory interface
type Yaml struct {
	YamlFile string
}

// NewYamlInventory returns a Yaml type that implements the inventory interface.
func NewYamlInventory(yamlFile string) (Repository, error) {
	return &Yaml{YamlFile: yamlFile}, nil
}

// AssetByID returns device attributes by its identifier
func (c *Yaml) AssetByID(_ context.Context, _ string) (*model.Asset, error) {
	return nil, nil
}
