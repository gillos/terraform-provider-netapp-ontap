package interfaces

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mitchellh/mapstructure"
	"github.com/netapp/terraform-provider-netapp-ontap/internal/restclient"
	"github.com/netapp/terraform-provider-netapp-ontap/internal/utils"
)

// IPRouteGetDataModelONTAP describes the GET record data model using go types for mapping.
type IPRouteGetDataModelONTAP struct {
	Destination DestinationDataSourceModel `mapstructure:"destination"`
	UUID        string                     `mapstructure:"uuid"`
	Gateway     string                     `mapstructure:"gateway"`
	Metric      int64                      `mapstructure:"metric"`
	SVMName     Vserver                    `mapstructure:"svm"`
}

// IPRouteResourceBodyDataModelONTAP describes the body data model using go types for mapping.
type IPRouteResourceBodyDataModelONTAP struct {
	Destination DestinationDataSourceModel `mapstructure:"destination"`
	SVM         Vserver                    `mapstructure:"svm"`
	Gateway     string                     `mapstructure:"gateway"`
	Metric      int64                      `mapstructure:"metric"`
}

// DestinationDataSourceModel describes the GET record data model using go types for mapping.
type DestinationDataSourceModel struct {
	Address string `mapstructure:"address"`
	Netmask string `mapstructure:"netmask"`
}

// GetIPRoute to get net_route info
func GetIPRoute(errorHandler *utils.ErrorHandler, r restclient.RestClient, Destination string, svmName string, version versionModelONTAP) (*IPRouteGetDataModelONTAP, error) {
	api := "/network/ip/routes"
	query := r.NewQuery()
	query.Set("destination.address", Destination)
	if svmName == "" {
		query.Set("scope", "cluster")
	} else {
		query.Set("svm.name", svmName)
		query.Set("scope", "svm")
	}
	var fields = []string{"destination", "svm.name", "gateway", "scope"}
	if version.Generation == 9 && version.Major > 10 {
		fields = append(fields, "metric")
	}
	query.Fields(fields)
	statusCode, response, err := r.GetNilOrOneRecord(api, query, nil)
	if err == nil && response == nil {
		err = fmt.Errorf("no response for GET %s", api)
	}
	if err != nil {
		return nil, errorHandler.MakeAndReportError("error reading /network/ip/routes info", fmt.Sprintf("error on GET %s: %s, statusCode %d", api, err, statusCode))
	}

	var dataONTAP IPRouteGetDataModelONTAP
	if err := mapstructure.Decode(response, &dataONTAP); err != nil {
		return nil, errorHandler.MakeAndReportError(fmt.Sprintf("failed to decode response from GET %s", api),
			fmt.Sprintf("error: %s, statusCode %d, response %#v", err, statusCode, response))
	}
	tflog.Debug(errorHandler.Ctx, fmt.Sprintf("Read /network/ip/routes data source: %#v", dataONTAP))
	return &dataONTAP, nil
}

// CreateIPRoute to create net_route
func CreateIPRoute(errorHandler *utils.ErrorHandler, r restclient.RestClient, body IPRouteResourceBodyDataModelONTAP) (*IPRouteGetDataModelONTAP, error) {
	api := "/network/ip/routes"
	var bodyMap map[string]interface{}
	if err := mapstructure.Decode(body, &bodyMap); err != nil {
		return nil, errorHandler.MakeAndReportError("error encoding /network/ip/routes body", fmt.Sprintf("error on encoding %s body: %s, body: %#v", api, err, body))
	}
	query := r.NewQuery()
	query.Add("return_records", "true")
	statusCode, response, err := r.CallCreateMethod(api, query, bodyMap)
	if err != nil {
		return nil, errorHandler.MakeAndReportError("error creating /network/ip/routes", fmt.Sprintf("error on POST %s: %s, statusCode %d", api, err, statusCode))
	}

	var dataONTAP IPRouteGetDataModelONTAP
	if err := mapstructure.Decode(response.Records[0], &dataONTAP); err != nil {
		return nil, errorHandler.MakeAndReportError("error decoding /network/ip/routes info", fmt.Sprintf("error on decode /network/ip/routes info: %s, statusCode %d, response %#v", err, statusCode, response))
	}
	tflog.Debug(errorHandler.Ctx, fmt.Sprintf("Create /network/ip/routes source - udata: %#v", dataONTAP))
	return &dataONTAP, nil
}

// DeleteIPRoute to delete net_route
func DeleteIPRoute(errorHandler *utils.ErrorHandler, r restclient.RestClient, uuid string) error {
	api := "/network/ip/routes"
	statusCode, _, err := r.CallDeleteMethod(api+"/"+uuid, nil, nil)
	if err != nil {
		return errorHandler.MakeAndReportError("error deleting /network/ip/routes", fmt.Sprintf("error on DELETE %s: %s, statusCode %d", api, err, statusCode))
	}
	return nil
}
