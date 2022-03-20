# Azure Keyvault Exporter

[![license](https://img.shields.io/github/license/webdevops/azure-keyvault-exporter.svg)](https://github.com/webdevops/azure-keyvault-exporter/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fazure--keyvault--exporter-blue)](https://hub.docker.com/r/webdevops/azure-keyvault-exporter/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fazure--keyvault--exporter-blue)](https://quay.io/repository/webdevops/azure-keyvault-exporter)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/azure-keyvault-exporter)](https://artifacthub.io/packages/search?repo=azure-keyvault-exporter)

Prometheus exporter for Azure Keyvaults which exports general information and dates (created, updated, notBefore, expiry) for secrets, keys and certificates.

## Configuration

Normally no configuration is needed but can be customized using environment variables.

```
Usage:
  azure-keyvault-exporter [OPTIONS]

Application Options:
      --debug                debug mode [$DEBUG]
  -v, --verbose              verbose mode [$VERBOSE]
      --log.json             Switch log output to json format [$LOG_JSON]
      --azure.environment=   Azure environment name (default: AZUREPUBLICCLOUD) [$AZURE_ENVIRONMENT]
      --azure.subscription=  Azure subscription ID (space delimiter) [$AZURE_SUBSCRIPTION_ID]
      --azure.location=      Azure locations (space delimiter) (default: westeurope, northeurope) [$AZURE_LOCATION]
      --azure.resourcegroup= Azure ResourceGroup [$AZURE_RESOURCEGROUP]
      --azure.resource-tag=  Azure Resource tags (space delimiter) (default: owner) [$AZURE_RESOURCE_TAG]
      --scrape.time=         Default scrape time (time.duration) (default: 5m) [$SCRAPE_TIME]
      --bind=                Server address (default: :8080) [$SERVER_BIND]

Help Options:
  -h, --help                 Show this help message

```

for Azure API authentication (using ENV vars) see https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication

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

Tag can be dynamically added to metrics and processed though filters

format is: `tagname?filter1` or `tagname?filter1&filter2`

| Tag filter | Description                 |
|------------|-----------------------------|
| `toLower`  | Lowercasing Azure tag value |
| `toUpper`  | Uppercasing Azure tag value |

### AzureTracing metrics

(with 22.2.0 and later)

Azuretracing metrics collects latency and latency from azure-sdk-for-go and creates metrics and is controllable using
environment variables (eg. setting buckets, disabling metrics or disable autoreset).

| Metric                                   | Description                                                                            |
|------------------------------------------|----------------------------------------------------------------------------------------|
| `azurerm_api_ratelimit`                  | Azure ratelimit metrics (only on /metrics, resets after query due to limited validity) |
| `azurerm_api_request_*`                  | Azure request count and latency as histogram                                           |

#### Settings

| Environment variable                     | Example                            | Description                                                    |
|------------------------------------------|------------------------------------|----------------------------------------------------------------|
| `METRIC_AZURERM_API_REQUEST_BUCKETS`     | `1, 2.5, 5, 10, 30, 60, 90, 120`   | Sets buckets for `azurerm_api_request` histogram metric        |
| `METRIC_AZURERM_API_REQUEST_ENABLE`      | `false`                            | Enables/disables `azurerm_api_request_*` metric                |
| `METRIC_AZURERM_API_REQUEST_LABELS`      | `apiEndpoint, method, statusCode`  | Controls labels of `azurerm_api_request_*` metric              |
| `METRIC_AZURERM_API_RATELIMIT_ENABLE`    | `false`                            | Enables/disables `azurerm_api_ratelimit` metric                |
| `METRIC_AZURERM_API_RATELIMIT_AUTORESET` | `false`                            | Enables/disables `azurerm_api_ratelimit` autoreset after fetch |


| `azurerm_api_request` label | Status             | Description                                                                                              |
|-----------------------------|--------------------|----------------------------------------------------------------------------------------------------------|
| `apiEndpoint`               | enabled by default | hostname of endpoint (max 3 parts)                                                                       |
| `routingRegion`             | enabled by default | detected region for API call, either routing region from Azure Management API or Azure resource location |
| `subscriptionID`            | enabled by default | detected subscriptionID                                                                                  |
| `tenantID`                  | enabled by default | detected tenantID (extracted from jwt auth token)                                                        |
| `resourceProvider`          | enabled by default | detected Azure Management API provider                                                                   |
| `method`                    | enabled by default | HTTP method                                                                                              |
| `statusCode`                | enabled by default | HTTP status code                                                                                         |
