# Azure Keyvault Exporter

[![license](https://img.shields.io/github/license/webdevops/azure-keyvault-exporter.svg)](https://github.com/webdevops/azure-keyvault-exporter/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fazure--keyvault--exporter-blue)](https://hub.docker.com/r/webdevops/azure-keyvault-exporter/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fazure--keyvault--exporter-blue)](https://quay.io/repository/webdevops/azure-keyvault-exporter)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/azure-keyvault-exporter)](https://artifacthub.io/packages/search?repo=azure-keyvault-exporter)

Prometheus exporter for Azure KeyVaults which exports general information and dates (created, updated, notBefore, expiry) for secrets, keys and certificates.

## Configuration

```
Usage:
  azure-keyvault-exporter [OPTIONS]

Application Options:
      --log.debug             debug mode [$LOG_DEBUG]
      --log.devel             development mode [$LOG_DEVEL]
      --log.json              Switch log output to json format [$LOG_JSON]
      --azure.environment=    Azure environment name (default: AZUREPUBLICCLOUD) [$AZURE_ENVIRONMENT]
      --azure.subscription=   Azure subscription ID (space delimiter) [$AZURE_SUBSCRIPTION_ID]
      --azure.resource-tag=   Azure Resource tags (space delimiter) (default: owner) [$AZURE_RESOURCE_TAG]
      --keyvault.filter=      Filter KeyVaults via ResourceGraph kusto filter, query: 'resource | ${filter} | project id' [$KEYVAULT_FILTER]
      --keyvault.content.tag= KeyVault content (secret, key, certificates) tags (space delimiter) [$KEYVAULT_CONTENT_TAG]
      --cache.path=           Cache path (to folder, file://path... or azblob://storageaccount.blob.core.windows.net/containername)
                              [$CACHE_PATH]
      --scrape.time=          Default scrape time (time.duration) (default: 5m) [$SCRAPE_TIME]
      --scrape.concurrency=   Defines who many Keyvaults can be scraped at the same time (default: 10) [$SCRAPE_CONCURRENCY]
      --server.bind=          Server address (default: :8080) [$SERVER_BIND]
      --server.timeout.read=  Server read timeout (default: 5s) [$SERVER_TIMEOUT_READ]
      --server.timeout.write= Server write timeout (default: 10s) [$SERVER_TIMEOUT_WRITE]

Help Options:
  -h, --help                  Show this help message
```

for Azure API authentication (using ENV vars) see following documentations:
- https://github.com/webdevops/go-common/blob/main/azuresdk/README.md
- https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication

## Metrics

| Metric                                 | Description                                                         |
|----------------------------------------|---------------------------------------------------------------------|
| `azurerm_keyvault_info`                | Azure KeyVault information                                          |
| `azurerm_keyvault_status`              | Azure KeyVault status information (eg. if accessable from exporter) |
| `azurerm_keyvault_entries`             | Count of entries (seperated by type) inside Azure KeyVault          |
| `azurerm_keyvault_key_info`            | General inforamtions about keys                                     |
| `azurerm_keyvault_key_status`          | Status information (notBefore & expiry date)                        |
| `azurerm_keyvault_secret_info`         | General inforamtions about secrets                                  |
| `azurerm_keyvault_secret_status`       | Status information (notBefore & expiry date)                        |
| `azurerm_keyvault_certificate_info`    | General inforamtions about certificate                              |
| `azurerm_keyvault_certificate_status`  | Status information (notBefore & expiry date)                        |

### ResourceTags handling

see [armclient tagmanager documentation](https://github.com/webdevops/go-common/blob/main/azuresdk/README.md#tag-manager)

### AzureTracing metrics

see [armclient tracing documentation](https://github.com/webdevops/go-common/blob/main/azuresdk/README.md#azuretracing-metrics)

### Caching

see [prometheus collector cache documentation](https://github.com/webdevops/go-common/blob/main/prometheus/README.md#caching)

