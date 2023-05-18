package config

import (
	"encoding/json"
	"time"
)

type (
	Opts struct {
		// logger
		Logger struct {
			Debug       bool `long:"log.debug"    env:"LOG_DEBUG"  description:"debug mode"`
			Development bool `long:"log.devel"    env:"LOG_DEVEL"  description:"development mode"`
			Json        bool `long:"log.json"     env:"LOG_JSON"   description:"Switch log output to json format"`
		}

		// azure
		Azure struct {
			Environment  *string  `long:"azure.environment"       env:"AZURE_ENVIRONMENT"                        description:"Azure environment name" default:"AZUREPUBLICCLOUD"`
			Subscription []string `long:"azure.subscription"      env:"AZURE_SUBSCRIPTION_ID"     env-delim:" "  description:"Azure subscription ID (space delimiter)"`
			ResourceTags []string `long:"azure.resource-tag"      env:"AZURE_RESOURCE_TAG"        env-delim:" "  description:"Azure Resource tags (space delimiter)"                              default:"owner"`
		}

		KeyVault struct {
			Filter string `long:"keyvault.filter"   env:"KEYVAULT_FILTER"   description:"Filter KeyVaults via ResourceGraph kusto filter, query: 'resource | ${filter} | project id'"`
		}

		// caching
		Cache struct {
			Path string `long:"cache.path" env:"CACHE_PATH" description:"Cache path (to folder, file://path... or azblob://storageaccount.blob.core.windows.net/containername)"`
		}

		// scrape times
		Scrape struct {
			Time        time.Duration `long:"scrape.time"         env:"SCRAPE_TIME"         description:"Default scrape time (time.duration)"                         default:"5m"`
			Concurrency int           `long:"scrape.concurrency"  env:"SCRAPE_CONCURRENCY"  description:"Defines who many Keyvaults can be scraped at the same time"  default:"10"`
		}

		Server struct {
			// general options
			Bind         string        `long:"server.bind"              env:"SERVER_BIND"           description:"Server address"        default:":8080"`
			ReadTimeout  time.Duration `long:"server.timeout.read"      env:"SERVER_TIMEOUT_READ"   description:"Server read timeout"   default:"5s"`
			WriteTimeout time.Duration `long:"server.timeout.write"     env:"SERVER_TIMEOUT_WRITE"  description:"Server write timeout"  default:"10s"`
		}
	}
)

func (o *Opts) GetCachePath(path string) (ret *string) {
	if o.Cache.Path != "" {
		tmp := o.Cache.Path + "/" + path
		ret = &tmp
	}

	return
}

func (o *Opts) GetJson() []byte {
	jsonBytes, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return jsonBytes
}
