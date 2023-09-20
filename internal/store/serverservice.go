package store

import (
	"context"
	"net/url"
	"time"

	sservice "go.hollow.sh/serverservice/pkg/api/v1"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/coreos/go-oidc"
	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/metal-toolbox/disko/internal/app"
	"github.com/metal-toolbox/disko/internal/metrics"
	"github.com/metal-toolbox/disko/internal/model"
	"github.com/pkg/errors"
)

const (
	// connectionTimeout is the maximum amount of time spent on each http connection to serverservice.
	connectionTimeout = 30 * time.Second

	pkgName = "internal/store"
)

var (
	ErrNoAttributes          = errors.New("no disko attribute found")
	ErrAttributeList         = errors.New("error in serverservice disko attribute list")
	ErrAttributeCreate       = errors.New("error in serverservice disko attribute create")
	ErrAttributeUpdate       = errors.New("error in serverservice disko attribute update")
	ErrVendorModelAttributes = errors.New("device vendor, model attributes not found in serverservice")
	ErrDeviceStatus          = errors.New("error serverservice device status")

	ErrDeviceID = errors.New("device UUID error")

	// ErrBMCAddress is returned when an error occurs in the BMC address lookup.
	ErrBMCAddress = errors.New("error in server BMC Address")

	// ErrDeviceState is returned when an error occurs in the device state  lookup.
	ErrDeviceState = errors.New("error in device state")

	// ErrServerserviceAttrObj is retuned when an error occurred in unpacking the attribute.
	ErrServerserviceAttrObj = errors.New("serverservice attribute error")

	// ErrServerserviceVersionedAttrObj is retuned when an error occurred in unpacking the versioned attribute.
	ErrServerserviceVersionedAttrObj = errors.New("serverservice versioned attribute error")

	// ErrServerserviceQuery is returned when a server service query fails.
	ErrServerserviceQuery = errors.New("serverservice query returned error")

	ErrFirmwareSetLookup = errors.New("firmware set error")
)

type Serverservice struct {
	config *app.ServerserviceOptions
	client *sservice.Client
	logger *logrus.Logger
}

func NewServerserviceStore(ctx context.Context, config *app.ServerserviceOptions, logger *logrus.Logger) (Repository, error) {
	var client *sservice.Client
	var err error

	if !config.DisableOAuth {
		client, err = newClientWithOAuth(ctx, config, logger)
		if err != nil {
			return nil, err
		}
	} else {
		client, err = sservice.NewClientWithToken("fake", config.Endpoint, nil)
		if err != nil {
			return nil, err
		}
	}

	serverservice := &Serverservice{
		client: client,
		config: config,
		logger: logger,
	}

	return serverservice, nil
}

// returns a serverservice retryable http client with Otel and Oauth wrapped in
func newClientWithOAuth(ctx context.Context, cfg *app.ServerserviceOptions, logger *logrus.Logger) (*sservice.Client, error) {
	// init retryable http client
	retryableClient := retryablehttp.NewClient()

	// set retryable HTTP client to be the otel http client to collect telemetry
	retryableClient.HTTPClient = otelhttp.DefaultClient

	// disable default debug logging on the retryable client
	if logger.Level < logrus.DebugLevel {
		retryableClient.Logger = nil
	} else {
		retryableClient.Logger = logger
	}

	// setup oidc provider
	provider, err := oidc.NewProvider(ctx, cfg.OidcIssuerEndpoint)
	if err != nil {
		return nil, err
	}

	clientID := model.AppName

	if cfg.OidcClientID != "" {
		clientID = cfg.OidcClientID
	}

	// setup oauth configuration
	oauthConfig := clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   cfg.OidcClientSecret,
		TokenURL:       provider.Endpoint().TokenURL,
		Scopes:         cfg.OidcClientScopes,
		EndpointParams: url.Values{"audience": []string{cfg.OidcAudienceEndpoint}},
	}

	// wrap OAuth transport, cookie jar in the retryable client
	oAuthclient := oauthConfig.Client(ctx)

	retryableClient.HTTPClient.Transport = oAuthclient.Transport
	retryableClient.HTTPClient.Jar = oAuthclient.Jar

	httpClient := retryableClient.StandardClient()
	httpClient.Timeout = connectionTimeout

	return sservice.NewClientWithToken(
		cfg.OidcClientSecret,
		cfg.Endpoint,
		httpClient,
	)
}

func (s *Serverservice) registerMetric(queryKind string) {
	metrics.StoreQueryErrorCount.With(
		prometheus.Labels{
			"storeKind": "serverservice",
			"queryKind": queryKind,
		},
	).Inc()
}

// AssetByID returns an Asset object with various attributes populated.
func (s *Serverservice) AssetByID(ctx context.Context, id string) (*model.Asset, error) {
	ctx, span := otel.Tracer(pkgName).Start(ctx, "Serverservice.AssetByID")
	defer span.End()

	deviceUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.Wrap(ErrDeviceID, err.Error()+id)
	}

	asset := &model.Asset{ID: deviceUUID}

	// query credentials
	credential, _, err := s.client.GetCredential(ctx, deviceUUID, sservice.ServerCredentialTypeBMC)
	if err != nil {
		s.registerMetric("GetCredential")

		return nil, errors.Wrap(ErrServerserviceQuery, "GetCredential: "+err.Error())
	}

	asset.BmcUsername = credential.Username
	asset.BmcPassword = credential.Password

	// query the server object
	srv, _, err := s.client.Get(ctx, deviceUUID)
	if err != nil {
		s.registerMetric("GetServer")

		return nil, errors.Wrap(ErrServerserviceQuery, "GetServer: "+err.Error())
	}

	asset.FacilityCode = srv.FacilityCode

	// set bmc address
	asset.BmcAddress, err = s.bmcAddressFromAttributes(srv.Attributes)
	if err != nil {
		return nil, err
	}

	// set asset vendor attributes
	asset.Vendor, asset.Model, asset.Serial, err = s.vendorModelFromAttributes(srv.Attributes)
	if err != nil {
		s.logger.WithError(err).Warn(ErrVendorModelAttributes)
	}

	return asset, nil
}
