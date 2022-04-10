package main

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	keyvaultMgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	azureCommon "github.com/webdevops/go-common/azure"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	"github.com/webdevops/go-common/prometheus/collector"
)

type MetricsCollectorKeyvault struct {
	collector.Processor

	collector *collector.Collector

	keyvaultAuth autorest.Authorizer

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
	var err error
	m.collector = collector

	m.keyvaultAuth, err = auth.NewAuthorizerFromEnvironmentWithResource(AzureClient.Environment.ResourceIdentifiers.KeyVault)
	if err != nil {
		panic(err)
	}

	m.prometheus.keyvault = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_info",
			Help: "Azure KeyVault information",
		},
		azureCommon.AddResourceTagsToPrometheusLabelsDefinition(
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
	err := AzureSubscriptionsIterator.ForEachAsync(func(subscription subscriptions.Subscription) {
		contextLogger := m.collector.Logger.WithField("subscription", *subscription.SubscriptionID)
		contextLogger.Debug("collecting for subscription")
		m.collectSubscription(callback, subscription, contextLogger)
	})
	if err != nil {
		m.collector.Logger.Panic(err)
	}
}

func (m *MetricsCollectorKeyvault) collectSubscription(callback chan<- func(), subscription subscriptions.Subscription, logger *log.Entry) {
	var keyvaultResult keyvaultMgmt.VaultListResultIterator
	var err error

	ctx := m.collector.Context

	keyvaultClient := keyvaultMgmt.NewVaultsClientWithBaseURI(AzureClient.Environment.ResourceManagerEndpoint, *subscription.SubscriptionID)
	AzureClient.DecorateAzureAutorest(&keyvaultClient.BaseClient.Client)

	if opts.Azure.ResourceGroup != "" {
		keyvaultResult, err = keyvaultClient.ListByResourceGroupComplete(ctx, opts.Azure.ResourceGroup, nil)
	} else {
		keyvaultResult, err = keyvaultClient.ListBySubscriptionComplete(ctx, nil)
	}

	if err != nil {
		log.WithField("subscription", *subscription.SubscriptionID).Panic(err)
	}

	keyvaultCount := 0
	for keyvaultResult.NotDone() {
		vault := keyvaultResult.Value()

		azureResource, _ := azureCommon.ParseResourceId(*vault.ID)

		contextLogger := logger.WithFields(log.Fields{
			"keyvault":      azureResource.ResourceName,
			"location":      to.String(vault.Location),
			"resourceGroup": azureResource.ResourceGroup,
		})

		client := keyvault.New()
		AzureClient.DecorateAzureAutorestWithAuthorizer(&client.Client, m.keyvaultAuth)

		m.collector.WaitGroup.Add()
		go func(client keyvault.BaseClient, vault keyvaultMgmt.Vault, contextLogger *log.Entry) {
			defer m.collector.WaitGroup.Done()
			contextLogger.Info("collecting keyvault metrics")
			m.collectKeyVault(callback, client, vault, contextLogger)
		}(client, vault, contextLogger)

		if keyvaultResult.NextWithContext(ctx) != nil {
			break
		}

		keyvaultCount++
	}
}

func (m *MetricsCollectorKeyvault) collectKeyVault(callback chan<- func(), client keyvault.BaseClient, vault keyvaultMgmt.Vault, logger *log.Entry) (status bool) {
	ctx := m.collector.Context
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

	vaultResourceId := stringPtrToStringLower(vault.ID)

	azureResource, _ := azureCommon.ParseResourceId(*vault.ID)

	entrySecretsCount := float64(0)
	entryKeysCount := float64(0)
	entryCertsCount := float64(0)

	// ########################
	// Keyvault
	// ########################

	vaultLabels := prometheus.Labels{
		"subscriptionID": azureResource.Subscription,
		"resourceID":     vaultResourceId,
		"vaultName":      azureResource.ResourceName,
		"location":       to.String(vault.Location),
		"resourceGroup":  azureResource.ResourceGroup,
	}
	vaultLabels = azureCommon.AddResourceTagsToPrometheusLabels(vaultLabels, vault.Tags, opts.Azure.ResourceTags)
	vaultMetrics.AddInfo(vaultLabels)

	// ########################
	// Keys
	// ########################

	keyResult, err := client.GetKeysComplete(ctx, vaultUrl, nil)
	if err != nil {
		logger.Warn(err)
		vaultStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "access",
			"scope":      "keys",
		}, 0)
	} else {
		vaultStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "access",
			"scope":      "keys",
		}, 1)
	}

	for keyResult.NotDone() {
		item := keyResult.Value()
		entryKeysCount++

		vaultKeyMetrics.AddInfo(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"keyName":    parseKeyvaultObjectName(to.String(item.Kid)),
			"keyID":      to.String(item.Kid),
			"enabled":    boolToString(*item.Attributes.Enabled),
		})

		// expiry date
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"keyID":      to.String(item.Kid),
			"type":       "expiry",
		}, expiryDate)

		// not before
		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"keyID":      to.String(item.Kid),
			"type":       "notBefore",
		}, notBeforeDate)

		// created
		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"keyID":      to.String(item.Kid),
			"type":       "created",
		}, createdDate)

		// updated
		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"keyID":      to.String(item.Kid),
			"type":       "updated",
		}, updatedDate)

		if keyResult.NextWithContext(ctx) != nil {
			break
		}
	}

	// ########################
	// Secrets
	// ########################

	secretsResult, err := client.GetSecretsComplete(ctx, vaultUrl, nil)
	if err != nil {
		logger.Warn(err)
		vaultStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "access",
			"scope":      "secrets",
		}, 0)
	} else {
		vaultStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "access",
			"scope":      "secrets",
		}, 1)
	}

	for secretsResult.NotDone() {
		item := secretsResult.Value()
		entrySecretsCount++

		vaultSecretMetrics.AddInfo(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"secretName": parseKeyvaultObjectName(to.String(item.ID)),
			"secretID":   to.String(item.ID),
			"enabled":    boolToString(to.Bool(item.Attributes.Enabled)),
		})

		// expiry date
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"secretID":   to.String(item.ID),
			"type":       "expiry",
		}, expiryDate)

		// notbefore
		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"secretID":   to.String(item.ID),
			"type":       "notBefore",
		}, notBeforeDate)

		// created
		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"secretID":   to.String(item.ID),
			"type":       "created",
		}, createdDate)

		// updated
		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"secretID":   to.String(item.ID),
			"type":       "updated",
		}, updatedDate)

		if secretsResult.NextWithContext(ctx) != nil {
			break
		}
	}

	// ########################
	// Certificate
	// ########################

	certificateResult, err := client.GetCertificatesComplete(ctx, vaultUrl, nil)
	if err != nil {
		logger.Warn(err)
		vaultStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "access",
			"scope":      "certificates",
		}, 0)
	} else {
		vaultStatusMetrics.Add(prometheus.Labels{
			"resourceID": vaultResourceId,
			"vaultName":  azureResource.ResourceName,
			"type":       "access",
			"scope":      "certificates",
		}, 1)
	}

	for certificateResult.NotDone() {
		item := certificateResult.Value()
		entryCertsCount++

		vaultCertificateMetrics.AddInfo(prometheus.Labels{
			"resourceID":      vaultResourceId,
			"vaultName":       azureResource.ResourceName,
			"certificateName": parseKeyvaultObjectName(to.String(item.ID)),
			"certificateID":   to.String(item.ID),
			"enabled":         boolToString(to.Bool(item.Attributes.Enabled)),
		})

		// expiry
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"resourceID":    vaultResourceId,
			"vaultName":     azureResource.ResourceName,
			"certificateID": to.String(item.ID),
			"type":          "expiry",
		}, expiryDate)

		// notBefore
		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"resourceID":    vaultResourceId,
			"vaultName":     azureResource.ResourceName,
			"certificateID": to.String(item.ID),
			"type":          "notBefore",
		}, notBeforeDate)

		// created
		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"resourceID":    vaultResourceId,
			"vaultName":     azureResource.ResourceName,
			"certificateID": to.String(item.ID),
			"type":          "created",
		}, createdDate)

		// updated
		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"resourceID":    vaultResourceId,
			"vaultName":     azureResource.ResourceName,
			"certificateID": to.String(item.ID),
			"type":          "updated",
		}, updatedDate)

		if certificateResult.NextWithContext(ctx) != nil {
			break
		}
	}

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
