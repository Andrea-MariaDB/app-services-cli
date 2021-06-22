package serviceregistry

import (
	"context"
	"errors"
	"net/http"

	srsmgmtv1 "github.com/redhat-developer/app-services-sdk-go/registrymgmt/apiv1/client"
)

func GetServiceRegistryByID(ctx context.Context, api srsmgmtv1.RegistriesApi, registryID string) (*srsmgmtv1.Registry, *http.Response, error) {
	request := api.GetRegistry(ctx, registryID)
	registry, _, err := request.Execute()
	if err != nil {
		return nil, nil, err
	}
	return &registry, nil, err
}

func GetServiceRegistryByName(ctx context.Context, api srsmgmtv1.RegistriesApi, name string) (*srsmgmtv1.Registry, *http.Response, error) {
	return nil, nil, errors.New("Not implemented. Only ID supported because of missing backend feature")
}
