package main

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/go-common/azuresdk/armclient"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	"github.com/webdevops/go-common/prometheus/collector"
	"github.com/webdevops/go-common/utils/to"
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
		armclient.AddResourceTagsToPrometheusLabelsDefinition(
			[]string{
				"subscriptionID",
				"resourceID",
				"vaultName",
				"location",
				"resourceGroup",
			},
			opts.Azure.ResourceTags,
		),
	)
	prometheus.MustRegister(m.prometheus.keyvault)

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
	prometheus.MustRegister(m.prometheus.keyvaultStatus)

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
	prometheus.MustRegister(m.prometheus.keyvaultEntryCount)

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
	prometheus.MustRegister(m.prometheus.keyvaultKeyInfo)

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
	prometheus.MustRegister(m.prometheus.keyvaultKeyStatus)

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
	prometheus.MustRegister(m.prometheus.keyvaultSecretInfo)

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
	prometheus.MustRegister(m.prometheus.keyvaultSecretStatus)

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
	prometheus.MustRegister(m.prometheus.keyvaultCertificateInfo)

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
	prometheus.MustRegister(m.prometheus.keyvaultCertificateStatus)
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

	err := AzureSubscriptionsIterator.ForEachAsync(m.Logger(), func(subscription *armsubscriptions.Subscription, logger *log.Entry) {
		m.collectSubscription(ctx, callback, subscription, logger)
	})
	if err != nil {
		m.Logger().Panic(err)
	}
}

func (m *MetricsCollectorKeyvault) collectSubscription(ctx context.Context, callback chan<- func(), subscription *armsubscriptions.Subscription, logger *log.Entry) {
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

				contextLogger := logger.WithFields(log.Fields{
					"keyvault":      azureResource.ResourceName,
					"location":      to.String(keyvault.Location),
					"resourceGroup": azureResource.ResourceGroup,
				})

				m.WaitGroup().Add()
				go func(keyvault *armkeyvault.Vault, contextLogger *log.Entry) {
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

				contextLogger := logger.WithFields(log.Fields{
					"keyvault":      azureResource.ResourceName,
					"location":      to.String(keyvault.Location),
					"resourceGroup": azureResource.ResourceGroup,
				})

				m.WaitGroup().Add()
				go func(keyvault *armkeyvault.Vault, contextLogger *log.Entry) {
					defer m.WaitGroup().Done()
					contextLogger.Info("collecting keyvault metrics")
					m.collectKeyVault(callback, keyvault, contextLogger)
				}(keyvault, contextLogger)
			}
		}
	}
}

func (m *MetricsCollectorKeyvault) collectKeyVault(callback chan<- func(), vault *armkeyvault.Vault, logger *log.Entry) (status bool) {
	status = true

	vaultMetrics := prometheusCommon.NewMetricsList()
	vaultStatusMetrics := prometheusCommon.NewMetricsList()
	vaultKeyMetrics := prometheusCommon.NewMetricsList()
	vaultKeyStatusMetrics := prometheusCommon.NewMetricsList()
	vaultSecretMetrics := prometheusCommon.NewMetricsList()
	vaultSecretStatusMetrics := prometheusCommon.NewMetricsList()
	vaultCertificateMetrics := prometheusCommon.NewMetricsList()
	vaultCertificateStatusMetrics := prometheusCommon.NewMetricsList()

	vaultUrl := to.String(vault.Properties.VaultURI)

	vaultResourceId := to.StringLower(vault.ID)

	azureResource, _ := armclient.ParseResourceId(*vault.ID)

	entrySecretsCount := float64(0)
	entryKeysCount := float64(0)
	entryCertsCount := float64(0)

	// ########################
	// Vault
	// ########################

	vaultLabels := prometheus.Labels{
		"subscriptionID": azureResource.Subscription,
		"resourceID":     to.StringLower(vault.ID),
		"vaultName":      azureResource.ResourceName,
		"location":       to.String(vault.Location),
		"resourceGroup":  azureResource.ResourceGroup,
	}
	vaultLabels = armclient.AddResourceTagsToPrometheusLabels(vaultLabels, vault.Tags, opts.Azure.ResourceTags)
	vaultMetrics.AddInfo(vaultLabels)

	// ########################
	// Keys
	// ########################

	keyClient := azkeys.NewClient(vaultUrl, AzureClient.GetCred(), AzureClient.NewAzCoreClientOptions())
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

	secretClient := azsecrets.NewClient(vaultUrl, AzureClient.GetCred(), AzureClient.NewAzCoreClientOptions())
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

	certificateClient := azcertificates.NewClient(vaultUrl, AzureClient.GetCred(), AzureClient.NewAzCoreClientOptions())
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

	// ########################
	// Processing
	// ########################

	callback <- func() {
		vaultMetrics.GaugeSet(m.prometheus.keyvault)
		vaultStatusMetrics.GaugeSet(m.prometheus.keyvaultStatus)
		vaultKeyMetrics.GaugeSet(m.prometheus.keyvaultKeyInfo)
		vaultKeyStatusMetrics.GaugeSet(m.prometheus.keyvaultKeyStatus)
		vaultSecretMetrics.GaugeSet(m.prometheus.keyvaultSecretInfo)
		vaultSecretStatusMetrics.GaugeSet(m.prometheus.keyvaultSecretStatus)
		vaultCertificateMetrics.GaugeSet(m.prometheus.keyvaultCertificateInfo)
		vaultCertificateStatusMetrics.GaugeSet(m.prometheus.keyvaultCertificateStatus)

		m.prometheus.keyvaultEntryCount.With(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "secrets",
		}).Set(entrySecretsCount)

		m.prometheus.keyvaultEntryCount.With(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "keys",
		}).Set(entryKeysCount)

		m.prometheus.keyvaultEntryCount.With(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "certificates",
		}).Set(entryCertsCount)
	}

	return
}
