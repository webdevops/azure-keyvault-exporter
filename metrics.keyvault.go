package main

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/webdevops/go-common/azuresdk/armclient"
	"github.com/webdevops/go-common/prometheus/collector"
	"github.com/webdevops/go-common/utils/to"
	"go.uber.org/zap"
)

type MetricsCollectorKeyvault struct {
	collector.Processor

	prometheus struct {
		// general
		keyvault             *prometheus.GaugeVec
		keyvaultStatus       *prometheus.GaugeVec
		keyvaultAccessPolicy *prometheus.GaugeVec
		keyvaultEntryCount   *prometheus.GaugeVec

		// key
		keyvaultKeyInfo   *prometheus.GaugeVec
		keyvaultKeyStatus *prometheus.GaugeVec

		// secret
		keyvaultSecretInfo   *prometheus.GaugeVec
		keyvaultSecretStatus *prometheus.GaugeVec

		// certs
		keyvaultCertificateInfo   *prometheus.GaugeVec
		keyvaultCertificateStatus *prometheus.GaugeVec
	}
}

func (m *MetricsCollectorKeyvault) Setup(collector *collector.Collector) {
	m.Processor.Setup(collector)

	m.prometheus.keyvault = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_info",
			Help: "Azure KeyVault information",
		},
		AzureResourceTagManager.AddToPrometheusLabels(
			[]string{
				"subscriptionID",
				"resourceID",
				"vaultName",
				"location",
				"resourceGroup",
			},
		),
	)
	m.Collector.RegisterMetricList("keyvault", m.prometheus.keyvault, true)

	m.prometheus.keyvaultStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_status",
			Help: "Azure KeyVault status",
		},
		[]string{
			"resourceID",
			"vaultName",
			"type",
			"scope",
		},
	)
	m.Collector.RegisterMetricList("keyvaultStatus", m.prometheus.keyvaultStatus, true)

	m.prometheus.keyvaultEntryCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_entries",
			Help: "Azure KeyVault entries",
		},
		[]string{
			"resourceID",
			"vaultName",
			"type",
		},
	)
	m.Collector.RegisterMetricList("keyvaultEntryCount", m.prometheus.keyvaultEntryCount, true)

	// ------------------------------------------
	// key
	m.prometheus.keyvaultKeyInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_key_info",
			Help: "Azure KeyVault key information",
		},
		[]string{
			"resourceID",
			"vaultName",
			"keyName",
			"keyID",
			"enabled",
		},
	)
	m.Collector.RegisterMetricList("keyvaultKeyInfo", m.prometheus.keyvaultKeyInfo, true)

	m.prometheus.keyvaultKeyStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_key_status",
			Help: "Azure KeyVault key status",
		},
		[]string{
			"resourceID",
			"vaultName",
			"keyID",
			"type",
		},
	)
	m.Collector.RegisterMetricList("keyvaultKeyStatus", m.prometheus.keyvaultKeyStatus, true)

	// ------------------------------------------
	// secret
	m.prometheus.keyvaultSecretInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_secret_info",
			Help: "Azure KeyVault secret information",
		},
		[]string{
			"resourceID",
			"vaultName",
			"secretName",
			"secretID",
			"enabled",
		},
	)
	m.Collector.RegisterMetricList("keyvaultSecretInfo", m.prometheus.keyvaultSecretInfo, true)

	m.prometheus.keyvaultSecretStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_secret_status",
			Help: "Azure KeyVault secret status",
		},
		[]string{
			"resourceID",
			"vaultName",
			"secretID",
			"type",
		},
	)
	m.Collector.RegisterMetricList("keyvaultSecretStatus", m.prometheus.keyvaultSecretStatus, true)

	// ------------------------------------------
	// certificate
	m.prometheus.keyvaultCertificateInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_certificate_info",
			Help: "Azure KeyVault certificate information",
		},
		[]string{
			"resourceID",
			"vaultName",
			"certificateName",
			"certificateID",
			"enabled",
		},
	)
	m.Collector.RegisterMetricList("keyvaultCertificateInfo", m.prometheus.keyvaultCertificateInfo, true)

	m.prometheus.keyvaultCertificateStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_certificate_status",
			Help: "Azure KeyVault certificate status",
		},
		[]string{
			"resourceID",
			"vaultName",
			"certificateID",
			"type",
		},
	)
	m.Collector.RegisterMetricList("keyvaultCertificateStatus", m.prometheus.keyvaultCertificateStatus, true)

}

func (m *MetricsCollectorKeyvault) Reset() {
	m.prometheus.keyvault.Reset()
	m.prometheus.keyvaultStatus.Reset()
	m.prometheus.keyvaultEntryCount.Reset()
	m.prometheus.keyvaultKeyInfo.Reset()
	m.prometheus.keyvaultKeyStatus.Reset()
	m.prometheus.keyvaultSecretInfo.Reset()
	m.prometheus.keyvaultSecretStatus.Reset()
	m.prometheus.keyvaultCertificateInfo.Reset()
	m.prometheus.keyvaultCertificateStatus.Reset()
}

func (m *MetricsCollectorKeyvault) Collect(callback chan<- func()) {
	ctx := m.Context()

	err := AzureSubscriptionsIterator.ForEachAsync(m.Logger(), func(subscription *armsubscriptions.Subscription, logger *zap.SugaredLogger) {
		m.collectSubscription(ctx, callback, subscription, logger)
	})
	if err != nil {
		m.Logger().Panic(err)
	}
}

func (m *MetricsCollectorKeyvault) collectSubscription(ctx context.Context, callback chan<- func(), subscription *armsubscriptions.Subscription, logger *zap.SugaredLogger) {
	var err error

	keyvaultClient, err := armkeyvault.NewVaultsClient(*subscription.SubscriptionID, AzureClient.GetCred(), AzureClient.NewArmClientOptions())
	if err != nil {
		logger.Panic(err)
	}

	if opts.Azure.ResourceGroup != "" {
		pager := keyvaultClient.NewListByResourceGroupPager(opts.Azure.ResourceGroup, nil)

		for pager.More() {
			result, err := pager.NextPage(m.Context())
			if err != nil {
				logger.Panic(err)
			}

			if result.Value == nil {
				continue
			}

			for _, row := range result.Value {
				keyvault := row

				azureResource, _ := armclient.ParseResourceId(*keyvault.ID)

				contextLogger := logger.With(
					zap.String("keyvault", azureResource.ResourceName),
					zap.String("location", to.String(keyvault.Location)),
					zap.String("resourceGroup", azureResource.ResourceGroup),
				)

				m.WaitGroup().Add()
				go func(keyvault *armkeyvault.Vault, contextLogger *zap.SugaredLogger) {
					defer m.WaitGroup().Done()
					contextLogger.Info("collecting keyvault metrics")
					m.collectKeyVault(callback, keyvault, contextLogger)
				}(keyvault, contextLogger)
			}
		}

	} else {
		pager := keyvaultClient.NewListBySubscriptionPager(nil)

		for pager.More() {
			result, err := pager.NextPage(m.Context())
			if err != nil {
				logger.Panic(err)
			}

			if result.Value == nil {
				continue
			}

			for _, row := range result.Value {
				keyvault := row

				azureResource, _ := armclient.ParseResourceId(*keyvault.ID)

				contextLogger := logger.With(
					zap.String("keyvault", azureResource.ResourceName),
					zap.String("location", to.String(keyvault.Location)),
					zap.String("resourceGroup", azureResource.ResourceGroup),
				)

				m.WaitGroup().Add()
				go func(keyvault *armkeyvault.Vault, contextLogger *zap.SugaredLogger) {
					defer m.WaitGroup().Done()
					contextLogger.Info("collecting keyvault metrics")
					m.collectKeyVault(callback, keyvault, contextLogger)
				}(keyvault, contextLogger)
			}
		}
	}
}

func (m *MetricsCollectorKeyvault) collectKeyVault(callback chan<- func(), vault *armkeyvault.Vault, logger *zap.SugaredLogger) (status bool) {
	status = true

	vaultMetrics := m.Collector.GetMetricList("keyvault")
	vaultStatusMetrics := m.Collector.GetMetricList("keyvaultStatus")
	vaultKeyMetrics := m.Collector.GetMetricList("keyvaultKeyInfo")
	vaultKeyStatusMetrics := m.Collector.GetMetricList("keyvaultKeyStatus")
	vaultSecretMetrics := m.Collector.GetMetricList("keyvaultSecretInfo")
	vaultSecretStatusMetrics := m.Collector.GetMetricList("keyvaultSecretStatus")
	vaultCertificateMetrics := m.Collector.GetMetricList("keyvaultCertificateInfo")
	vaultCertificateStatusMetrics := m.Collector.GetMetricList("keyvaultCertificateStatus")
	vaultEntryCountMetrics := m.Collector.GetMetricList("keyvaultEntryCount")

	vaultUrl := to.String(vault.Properties.VaultURI)

	vaultResourceId := to.StringLower(vault.ID)

	azureResource, _ := armclient.ParseResourceId(vaultResourceId)

	entrySecretsCount := float64(0)
	entryKeysCount := float64(0)
	entryCertsCount := float64(0)

	// ########################
	// Vault
	// ########################

	vaultLabels := prometheus.Labels{
		"subscriptionID": azureResource.Subscription,
		"resourceID":     vaultResourceId,
		"vaultName":      azureResource.ResourceName,
		"location":       to.String(vault.Location),
		"resourceGroup":  azureResource.ResourceGroup,
	}
	vaultLabels = AzureResourceTagManager.AddResourceTagsToPrometheusLabels(m.Context(), vaultLabels, vaultResourceId)
	vaultMetrics.AddInfo(vaultLabels)

	// ########################
	// Keys
	// ########################

	keyOpts := azkeys.ClientOptions{
		ClientOptions: *AzureClient.NewAzCoreClientOptions(),
	}
	keyClient, err := azkeys.NewClient(vaultUrl, AzureClient.GetCred(), &keyOpts)
	if err != nil {
		logger.Panic(err.Error())
	}
	keyPager := keyClient.NewListKeysPager(nil)

	keyStatus := float64(1)
	for keyPager.More() {
		result, err := keyPager.NextPage(m.Context())
		if err != nil {
			logger.Warn(err)
			keyStatus = 0
			break
		}

		if result.Value == nil {
			continue
		}

		for _, row := range result.Value {
			item := row
			entryKeysCount++

			itemID := string(*item.KID)
			itemName := item.KID.Name()

			vaultKeyMetrics.AddInfo(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"keyName":    itemName,
				"keyID":      itemID,
				"enabled":    to.BoolString(to.Bool(item.Attributes.Enabled)),
			})

			// expiry date
			expiryDate := float64(0)
			if item.Attributes.Expires != nil {
				expiryDate = float64(item.Attributes.Expires.Unix())
			}
			vaultKeyStatusMetrics.Add(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"keyID":      itemID,
				"type":       "expiry",
			}, expiryDate)

			// not before
			notBeforeDate := float64(0)
			if item.Attributes.NotBefore != nil {
				notBeforeDate = float64(item.Attributes.NotBefore.Unix())
			}
			vaultKeyStatusMetrics.Add(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"keyID":      itemID,
				"type":       "notBefore",
			}, notBeforeDate)

			// created
			createdDate := float64(0)
			if item.Attributes.Created != nil {
				createdDate = float64(item.Attributes.Created.Unix())
			}
			vaultKeyStatusMetrics.Add(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"keyID":      itemID,
				"type":       "created",
			}, createdDate)

			// updated
			updatedDate := float64(0)
			if item.Attributes.Updated != nil {
				updatedDate = float64(item.Attributes.Updated.Unix())
			}
			vaultKeyStatusMetrics.Add(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"keyID":      itemID,
				"type":       "updated",
			}, updatedDate)
		}
	}

	vaultStatusMetrics.Add(prometheus.Labels{
		"resourceID": vaultResourceId,
		"vaultName":  azureResource.ResourceName,
		"type":       "access",
		"scope":      "keys",
	}, keyStatus)

	// ########################
	// Secrets
	// ########################

	secretOpts := azsecrets.ClientOptions{
		ClientOptions: *AzureClient.NewAzCoreClientOptions(),
	}
	secretClient, err := azsecrets.NewClient(vaultUrl, AzureClient.GetCred(), &secretOpts)
	if err != nil {
		logger.Panic(err.Error())
	}
	secretPager := secretClient.NewListSecretsPager(nil)

	secretStatus := float64(1)
	for secretPager.More() {
		result, err := secretPager.NextPage(m.Context())
		if err != nil {
			logger.Warn(err)
			secretStatus = 0
			break
		}

		if result.Value == nil {
			continue
		}

		for _, row := range result.Value {
			item := row
			entrySecretsCount++

			itemID := string(*item.ID)
			itemName := item.ID.Name()

			vaultSecretMetrics.AddInfo(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"secretName": itemName,
				"secretID":   itemID,
				"enabled":    to.BoolString(to.Bool(item.Attributes.Enabled)),
			})

			// expiry date
			expiryDate := float64(0)
			if item.Attributes.Expires != nil {
				expiryDate = float64(item.Attributes.Expires.Unix())
			}
			vaultSecretStatusMetrics.Add(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"secretID":   itemID,
				"type":       "expiry",
			}, expiryDate)

			// notbefore
			notBeforeDate := float64(0)
			if item.Attributes.NotBefore != nil {
				notBeforeDate = float64(item.Attributes.NotBefore.Unix())
			}
			vaultSecretStatusMetrics.Add(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"secretID":   itemID,
				"type":       "notBefore",
			}, notBeforeDate)

			// created
			createdDate := float64(0)
			if item.Attributes.Created != nil {
				createdDate = float64(item.Attributes.Created.Unix())
			}
			vaultSecretStatusMetrics.Add(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"secretID":   itemID,
				"type":       "created",
			}, createdDate)

			// updated
			updatedDate := float64(0)
			if item.Attributes.Updated != nil {
				updatedDate = float64(item.Attributes.Updated.Unix())
			}
			vaultSecretStatusMetrics.Add(prometheus.Labels{
				"resourceID": vaultResourceId,
				"vaultName":  azureResource.ResourceName,
				"secretID":   itemID,
				"type":       "updated",
			}, updatedDate)
		}
	}

	vaultStatusMetrics.Add(prometheus.Labels{
		"resourceID": vaultResourceId,
		"vaultName":  azureResource.ResourceName,
		"type":       "access",
		"scope":      "secrets",
	}, secretStatus)

	// ########################
	// Certificate
	// ########################

	certificateOpts := azcertificates.ClientOptions{
		ClientOptions: *AzureClient.NewAzCoreClientOptions(),
	}
	certificateClient, err := azcertificates.NewClient(vaultUrl, AzureClient.GetCred(), &certificateOpts)
	if err != nil {
		logger.Panic(err.Error())
	}
	certificatePager := certificateClient.NewListCertificatesPager(nil)

	certificateStatus := float64(1)
	for certificatePager.More() {
		result, err := certificatePager.NextPage(m.Context())
		if err != nil {
			logger.Warn(err)
			certificateStatus = 0
			break
		}

		if result.Value == nil {
			continue
		}

		for _, row := range result.Value {
			item := row
			entryCertsCount++

			itemID := string(*item.ID)
			itemName := item.ID.Name()

			vaultCertificateMetrics.AddInfo(prometheus.Labels{
				"resourceID":      vaultResourceId,
				"vaultName":       azureResource.ResourceName,
				"certificateName": itemName,
				"certificateID":   itemID,
				"enabled":         to.BoolString(to.Bool(item.Attributes.Enabled)),
			})

			// expiry
			expiryDate := float64(0)
			if item.Attributes.Expires != nil {
				expiryDate = float64(item.Attributes.Expires.Unix())
			}
			vaultCertificateStatusMetrics.Add(prometheus.Labels{
				"resourceID":    vaultResourceId,
				"vaultName":     azureResource.ResourceName,
				"certificateID": itemID,
				"type":          "expiry",
			}, expiryDate)

			// notBefore
			notBeforeDate := float64(0)
			if item.Attributes.NotBefore != nil {
				notBeforeDate = float64(item.Attributes.NotBefore.Unix())
			}
			vaultCertificateStatusMetrics.Add(prometheus.Labels{
				"resourceID":    vaultResourceId,
				"vaultName":     azureResource.ResourceName,
				"certificateID": itemID,
				"type":          "notBefore",
			}, notBeforeDate)

			// created
			createdDate := float64(0)
			if item.Attributes.Created != nil {
				createdDate = float64(item.Attributes.Created.Unix())
			}
			vaultCertificateStatusMetrics.Add(prometheus.Labels{
				"resourceID":    vaultResourceId,
				"vaultName":     azureResource.ResourceName,
				"certificateID": itemID,
				"type":          "created",
			}, createdDate)

			// updated
			updatedDate := float64(0)
			if item.Attributes.Updated != nil {
				updatedDate = float64(item.Attributes.Updated.Unix())
			}
			vaultCertificateStatusMetrics.Add(prometheus.Labels{
				"resourceID":    vaultResourceId,
				"vaultName":     azureResource.ResourceName,
				"certificateID": itemID,
				"type":          "updated",
			}, updatedDate)

		}
	}

	vaultStatusMetrics.Add(prometheus.Labels{
		"resourceID": vaultResourceId,
		"vaultName":  azureResource.ResourceName,
		"type":       "access",
		"scope":      "certificates",
	}, certificateStatus)

	vaultEntryCountMetrics.Add(prometheus.Labels{
		"resourceID": vaultResourceId,
		"vaultName":  azureResource.ResourceName,
		"type":       "secrets",
	}, entrySecretsCount)

	vaultEntryCountMetrics.Add(prometheus.Labels{
		"resourceID": vaultResourceId,
		"vaultName":  azureResource.ResourceName,
		"type":       "keys",
	}, entryKeysCount)

	vaultEntryCountMetrics.Add(prometheus.Labels{
		"resourceID": vaultResourceId,
		"vaultName":  azureResource.ResourceName,
		"type":       "certificates",
	}, entryCertsCount)

	return
}
