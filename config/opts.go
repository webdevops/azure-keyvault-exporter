package config

import (
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
)

type (
	Opts struct {
		// logger
		Logger struct {
			Debug   bool `           long:"debug"        env:"DEBUG"    description:"debug mode"`
			Verbose bool `short:"v"  long:"verbose"      env:"VERBOSE"  description:"verbose mode"`
			LogJson bool `           long:"log.json"     env:"LOG_JSON" description:"Switch log output to json format"`
		}

		// azure
		Azure struct {
			Environment   *string  `long:"azure.environment"       env:"AZURE_ENVIRONMENT"                        description:"Azure environment name" default:"AZUREPUBLICCLOUD"`
			Subscription  []string `long:"azure.subscription"      env:"AZURE_SUBSCRIPTION_ID"     env-delim:" "  description:"Azure subscription ID"`
			Location      []string `long:"azure.location"          env:"AZURE_LOCATION"            env-delim:" "  description:"Azure locations"                                  default:"westeurope" default:"northeurope"` //nolint:staticcheck
			ResourceGroup string   `long:"azure.resourcegroup"     env:"AZURE_RESOURCEGROUP"                      description:"Azure ResourceGroup"`
			ResourceTags  []string `long:"azure.resource-tag"      env:"AZURE_RESOURCE_TAG"        env-delim:" "  description:"Azure Resource tags"                              default:"owner"`
		}

		Metrics struct {
			ResourceIdLowercase bool `long:"metrics.resourceid.lowercase"   env:"METRIC_RESOURCEID_LOWERCASE"       description:"Publish lowercase Azure Resoruce ID in metrics"`
		}

		// scrape times
		Scrape struct {
			Time time.Duration `long:"scrape.time"                    env:"SCRAPE_TIME"                    description:"Default scrape time (time.duration)"                      default:"5m"`
		}

		// general options
		ServerBind string `long:"bind"     env:"SERVER_BIND"   description:"Server address"     default:":8080"`
	}
)

func (o *Opts) GetJson() []byte {
	jsonBytes, err := json.Marshal(o)
	if err != nil {
		log.Panic(err)
	}
	return jsonBytes
}
