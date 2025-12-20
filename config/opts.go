package config

import (
	"encoding/json"
	"time"
)

type (
	Opts struct {
		// logger
		Logger struct {
			Level  string `long:"log.level"    env:"LOG_LEVEL"   description:"Log level" choice:"trace" choice:"debug" choice:"info" choice:"warning" choice:"error" default:"info"`                          // nolint:staticcheck // multiple choices are ok
			Format string `long:"log.format"   env:"LOG_FORMAT"  description:"Log format" choice:"logfmt" choice:"json" default:"logfmt"`                                                                     // nolint:staticcheck // multiple choices are ok
			Source string `long:"log.source"   env:"LOG_SOURCE"  description:"Show source for every log message (useful for debugging and bug reports)" choice:"" choice:"short" choice:"file" choice:"full"` // nolint:staticcheck // multiple choices are ok
			Color  string `long:"log.color"    env:"LOG_COLOR"   description:"Enable color for logs" choice:"" choice:"auto" choice:"yes" choice:"no"`                                                        // nolint:staticcheck // multiple choices are ok
			Time   bool   `long:"log.time"     env:"LOG_TIME"    description:"Show log time"`
		}

		// azure
		Azure struct {
			Environment  *string  `long:"azure.environment"       env:"AZURE_ENVIRONMENT"                        description:"Azure environment name" default:"AZUREPUBLICCLOUD"`
			Subscription []string `long:"azure.subscription"      env:"AZURE_SUBSCRIPTION_ID"     env-delim:" "  description:"Azure subscription ID (space delimiter)"`
			ResourceTags []string `long:"azure.resource-tag"      env:"AZURE_RESOURCE_TAG"        env-delim:" "  description:"Azure Resource tags (space delimiter)"                              default:"owner"`
		}

		KeyVault struct {
			Filter  string `long:"keyvault.filter"   env:"KEYVAULT_FILTER"   description:"Filter KeyVaults via ResourceGraph kusto filter, query: 'resource | ${filter} | project id'"`
			Content struct {
				Tags []string `long:"keyvault.content.tag"      env:"KEYVAULT_CONTENT_TAG"        env-delim:" "  description:"KeyVault content (secret, key, certificates) tags (space delimiter)"`
			}
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
