package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	simplejson "github.com/bitly/go-simplejson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	PLUGIN_NAME = "serge1peshcoff-googlefit-datasource"
)

// newDatasource returns datasource.ServeOpts.
func newDatasource() datasource.ServeOpts {
	// creates a instance manager for your plugin. The function passed
	// into `NewInstanceManger` is called when the instance is created
	// for the first time or when a datasource configuration changed.
	im := datasource.NewInstanceManager(newDataSourceInstance)
	ds := &SampleDatasource{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
	}
}

// SampleDatasource is an example datasource used to scaffold
// new datasource plugins with an backend.
type SampleDatasource struct {
	// The instance manager can help with lifecycle management
	// of datasource instances in plugins. It's not a requirements
	// but a best practice that we recommend that you follow.
	im instancemgmt.InstanceManager
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifer).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (td *SampleDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := td.query(ctx, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	Format string `json:"format"`
}

func (td *SampleDatasource) query(ctx context.Context, query backend.DataQuery) backend.DataResponse {
	// Unmarshal the json into our queryModel
	var qm queryModel

	response := backend.DataResponse{}

	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		return response
	}

	// Log a warning if `Format` is empty.
	if qm.Format == "" {
		log.DefaultLogger.Warn("format is empty. defaulting to time series")
	}

	// create data frame response
	frame := data.NewFrame("response")

	// add the time dimension
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []time.Time{query.TimeRange.From, query.TimeRange.To}),
	)

	// add values
	frame.Fields = append(frame.Fields,
		data.NewField("values", nil, []int64{10, 20}),
	)

	// add the frames to the response
	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (td *SampleDatasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if val, ok := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["code"]; !ok || val == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Code is not provided. Please press \"Sign in with Google\"",
		}, nil
	}

	if val, ok := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["clientSecret"]; !ok || val == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Client secret is not provided",
		}, nil
	}

	jsonData, err := simplejson.NewJson([]byte(req.PluginContext.DataSourceInstanceSettings.JSONData))
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Could not parse JSON",
		}, nil
	}

	clientID := jsonData.Get("clientId").MustString()
	if clientID == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Client ID is not provided",
		}, nil
	}

	redirectURI := jsonData.Get("redirectURI").MustString()
	if redirectURI == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Redirect URI is not provided",
		}, nil
	}

	clientSecret := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["clientSecret"]
	code := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["code"]

	log.DefaultLogger.Debug("clientId", "clientId", clientID)
	log.DefaultLogger.Debug("clientSecret", "clientSecret", clientSecret)
	log.DefaultLogger.Debug("code", "code", code)
	log.DefaultLogger.Debug("redirectURI", "redirectURI", redirectURI)

	token, err := getToken(ctx, clientID, clientSecret, code, redirectURI)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Error authorizing",
		}, nil
	}

	saveTokenToFile(token, PLUGIN_NAME, req.PluginContext.DataSourceInstanceSettings.ID)

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Everything is okay. Please do not resave, as the auth code is invalidated already.",
	}, nil
}

func getToken(ctx context.Context, clientID string, clientSecret string, code string, redirectURI string) (*oauth2.Token, error) {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURI,
	}

	token, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	return token, err
}

func saveTokenToFile(token *oauth2.Token, appID string, datasourceID int64) error {
	pluginsDirEnv, exists := os.LookupEnv("PLUGINS_DIR")
	if !exists {
		pluginsDirEnv = "/var/lib/grafana/plugins"
	}
	pluginsDir := path.Join(pluginsDirEnv, appID, "cache")

	log.DefaultLogger.Info("Plugins dir", "dir", pluginsDir)

	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		log.DefaultLogger.Info("Plugins dir does not exist")
		os.Mkdir(pluginsDir, os.ModePerm)
	}

	cacheFile := path.Join(pluginsDir, strconv.FormatInt(int64(datasourceID), 10)+".json")
	tokenAsString, err := json.Marshal(token)
	if err != nil {
		return err
	}

	ioutil.WriteFile(cacheFile, tokenAsString, os.ModePerm)
	return nil
}

func getTokenFromFile(appID string, datasourceID int64) (*oauth2.Token, error) {
	pluginsDirEnv := os.Getenv("PLUGINS_DIR")
	pluginsDir := path.Join(pluginsDirEnv, appID, "cache")
	cacheFile := path.Join(pluginsDir, strconv.FormatInt(int64(datasourceID), 10)+".json")
	fileContents, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var token *oauth2.Token
	err = json.Unmarshal(fileContents, &token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

type instanceSettings struct {
	httpClient *http.Client
}

func newDataSourceInstance(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &instanceSettings{
		httpClient: &http.Client{},
	}, nil
}

func (s *instanceSettings) Dispose() {
	// Called before creatinga a new instance to allow plugin authors
	// to cleanup.
}
