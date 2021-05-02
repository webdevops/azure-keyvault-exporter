Azure Keyvault Exporter
=======================

[![license](https://img.shields.io/github/license/webdevops/azure-keyvault-exporter.svg)](https://github.com/webdevops/azure-keyvault-exporter/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fazure--keyvault--exporter-blue)](https://hub.docker.com/r/webdevops/azure-keyvault-exporter/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fazure--keyvault--exporter-blue)](https://quay.io/repository/webdevops/azure-keyvault-exporter)

Prometheus exporter for Azure Keyvaults which exports general information and dates (created, updated, notBefore, expiry) for secrets, keys and certificates.

Configuration
-------------

Normally no configuration is needed but can be customized using environment variables.

```
Usage:
  azure-keyvault-exporter [OPTIONS]

Application Options:
      --debug                debug mode [$DEBUG]
  -v, --verbose              verbose mode [$VERBOSE]
      --log.json             Switch log output to json format [$LOG_JSON]
      --azure.environment=   Azure environment name (default: AZUREPUBLICCLOUD) [$AZURE_ENVIRONMENT]
      --azure.subscription=  Azure subscription ID [$AZURE_SUBSCRIPTION_ID]
      --azure.location=      Azure locations (default: westeurope, northeurope) [$AZURE_LOCATION]
      --azure.resourcegroup= Azure ResourceGroup [$AZURE_RESOURCEGROUP]
      --azure.resource-tag=  Azure Resource tags (default: owner) [$AZURE_RESOURCE_TAG]
      --scrape.time=         Default scrape time (time.duration) (default: 5m) [$SCRAPE_TIME]
      --bind=                Server address (default: :8080) [$SERVER_BIND]

Help Options:
  -h, --help                 Show this help message
```

for Azure API authentication (using ENV vars) see https://github.com/Azure/azure-sdk-for-go#authentication

Metrics
-------

| Metric                                 | Description                                                                           |
|----------------------------------------|---------------------------------------------------------------------------------------|
| `azurerm_keyvault_info`                | Azure KeyVault information                                                            |
| `azurerm_keyvault_status`              | Azure KeyVault status information (eg. if accessable from exporter)                   |
| `azurerm_keyvault_key_info`            | General inforamtions about keys                                                       |
| `azurerm_keyvault_key_status`          | Status information (notBefore & expiry date)                                          |
| `azurerm_keyvault_secret_info`         | General inforamtions about secrets                                                    |
| `azurerm_keyvault_secret_status`       | Status information (notBefore & expiry date)                                          |
| `azurerm_keyvault_certificate_info`    | General inforamtions about certificate                                                |
| `azurerm_keyvault_certificate_status`  | Status information (notBefore & expiry date)                                          |
| `azurerm_ratelimit`                    | Azure API ratelimit information (from last api call)                                  |

