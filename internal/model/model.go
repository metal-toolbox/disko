package model

import (
	"net"

	"github.com/google/uuid"
)

type (
	AppKind   string
	StoreKind string
	// LogLevel is the logging level string.
	LogLevel string
)

const (
	AppName               = "disko"
	AppKindWorker AppKind = "worker"
	AppKindClient AppKind = "client"

	StoreKindServerservice StoreKind = "serverservice"

	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
	LogLevelTrace LogLevel = "trace"
)

// AppKinds returns the supported disko app kinds
func AppKinds() []AppKind { return []AppKind{AppKindWorker} }

// StoreKinds returns the supported asset inventory
func StoreKinds() []StoreKind {
	return []StoreKind{StoreKindServerservice}
}

// Asset holds attributes of a server retrieved from the inventory store.
//
// nolint:govet // fieldalignment struct is easier to read in the current format
type Asset struct {
	ID uuid.UUID

	// Device BMC attributes
	BmcAddress  net.IP
	BmcUsername string
	BmcPassword string

	// Manufacturer attributes
	Vendor string
	Model  string
	Serial string

	// Facility this Asset is hosted in.
	FacilityCode string
}
