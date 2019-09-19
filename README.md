Azure Keyvault Exporter
=======================

[![license](https://img.shields.io/github/license/webdevops/azure-keyvault-exporter.svg)](https://github.com/webdevops/azure-keyvault-exporter/blob/master/LICENSE)
[![Docker](https://img.shields.io/badge/docker-webdevops%2Fazure--keyvault--exporter-blue.svg?longCache=true&style=flat&logo=docker)](https://hub.docker.com/r/webdevops/azure-keyvault-exporter/)
[![Docker Build Status](https://img.shields.io/docker/build/webdevops/azure-keyvault-exporter.svg)](https://hub.docker.com/r/webdevops/azure-keyvault-exporter/)

Prometheus exporter for Azure Keyvaults which exports general informations and dates (created, updated, notBefore, expiry) for secrets, keys and certificates.

Configuration
-------------

Normally no configuration is needed but can be customized using environment variables.

| Environment variable              | DefaultValue                | Description                                                                  |
|-----------------------------------|-----------------------------|------------------------------------------------------------------------------|
| `AZURE_SUBSCRIPTION_ID`           | `empty`                     | Azure Subscription IDs (empty for auto lookup)                               |
| `AZURE_RESOURCEGROUP`             |                             | Azure ResourceGroup used for collecting informations (filter, empty for all) |
| `AZURE_KEYVAULT_TAG`              | `owner`                     | Azure Keyvault tag which should be collected (methods available eg. `owner:lower` will transform content lowercase, methods: `lower`, `upper`, `title`)  |
| `SCRAPE_TIME`                     | `3h`                        | Time (time.Duration) between API calls                                       |
| `SCRAPE_TIME_RAND`                | `15m`                       | Randomization time between scrapes                                           |
| `SERVER_BIND`                     | `:8080`                     | IP/Port binding                                                              |

for Azure API authentication (using ENV vars) see https://github.com/Azure/azure-sdk-for-go#authentication

Metrics
-------

| Metric                                 | Description                                                                           |
|----------------------------------------|---------------------------------------------------------------------------------------|
| `azurerm_keyvault_info`                | Azure KeyVault informations                                                           |
| `azurerm_keyvault_key_info`            | General inforamtions about keys                                                       |
| `azurerm_keyvault_key_status`          | Status informations (notBefore & expiry date)                                         |
| `azurerm_keyvault_secret_info`         | General inforamtions about secrets                                                    |
| `azurerm_keyvault_secret_status`       | Status informations (notBefore & expiry date)                                         |
| `azurerm_keyvault_certificate_info`    | General inforamtions about certificate                                                |
| `azurerm_keyvault_certificate_status`  | Status informations (notBefore & expiry date)                                         |


Install
-------
```bash
go get -u github.com/webdevops/azure-keyvault-exporter
```



