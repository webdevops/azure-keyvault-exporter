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
			Debug bool `long:"log.debug"    env:"LOG_DEBUG"  description:"debug mode"`
			Trace bool `long:"log.trace"    env:"LOG_TRACE"  description:"trace mode"`
			Json  bool `long:"log.json"     env:"LOG_JSON"   description:"Switch log output to json format"`
		}

		// azure
		Azure struct {
			Environment   *string  `long:"azure.environment"       env:"AZURE_ENVIRONMENT"                        description:"Azure environment name" default:"AZUREPUBLICCLOUD"`
			Subscription  []string `long:"azure.subscription"      env:"AZURE_SUBSCRIPTION_ID"     env-delim:" "  description:"Azure subscription ID (space delimiter)"`
			ResourceGroup string   `long:"azure.resourcegroup"     env:"AZURE_RESOURCEGROUP"                      description:"Azure ResourceGroup"`
			ResourceTags  []string `long:"azure.resource-tag"      env:"AZURE_RESOURCE_TAG"        env-delim:" "  description:"Azure Resource tags (space delimiter)"                              default:"owner"`
		}

		// scrape times
		Scrape struct {
			Time        time.Duration `long:"scrape.time"         env:"SCRAPE_TIME"         description:"Default scrape time (time.duration)"                         default:"5m"`
			Concurrency int           `long:"scrape.concurrency"  env:"SCRAPE_CONCURRENCY"  description:"Defines who many Keyvaults can be scraped at the same time"  default:"10"`
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
