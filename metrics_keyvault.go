package main

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	keyvaultMgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	prometheusCommon "github.com/webdevops/go-prometheus-common"
	"sync"
	"time"
)

type MetricsCollectorKeyvault struct {
	CollectorProcessorGeneral

	keyvaultAuth autorest.Authorizer

	prometheus struct {
		keyvault                  *prometheus.GaugeVec
		keyvaultStatus            *prometheus.GaugeVec
		keyvaultKeyInfo           *prometheus.GaugeVec
		keyvaultKeyStatus         *prometheus.GaugeVec
		keyvaultSecretInfo        *prometheus.GaugeVec
		keyvaultSecretStatus      *prometheus.GaugeVec
		keyvaultCertificateInfo   *prometheus.GaugeVec
		keyvaultCertificateStatus *prometheus.GaugeVec
	}
}

func (m *MetricsCollectorKeyvault) Setup(collector *CollectorGeneral) {
	var err error
	m.CollectorReference = collector

	m.keyvaultAuth, err = auth.NewAuthorizerFromEnvironmentWithResource(azureEnvironment.ResourceIdentifiers.KeyVault)
	if err != nil {
		panic(err)
	}

	m.prometheus.keyvault = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_info",
			Help: "Azure KeyVault informations",
		},
		append(
			[]string{
				"subscriptionID",
				"resourceID",
				"vaultName",
				"location",
				"resourceGroup",
			},
			azureKeyvaultTag.prometheusLabels...,
		),
	)
	prometheus.MustRegister(m.prometheus.keyvault)

	m.prometheus.keyvaultStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_status",
			Help: "Azure KeyVault status",
		},
		[]string{
			"subscriptionID",
			"resourceID",
			"vaultName",
			"type",
			"scope",
		},
	)
	prometheus.MustRegister(m.prometheus.keyvaultStatus)

	// ------------------------------------------
	// key
	m.prometheus.keyvaultKeyInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_key_info",
			Help: "Azure KeyVault key informations",
		},
		[]string{
			"vaultName",
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
			Help: "Azure KeyVault secret informations",
		},
		[]string{
			"vaultName",
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
			Help: "Azure KeyVault certificate informations",
		},
		[]string{
			"vaultName",
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
	m.prometheus.keyvaultKeyInfo.Reset()
	m.prometheus.keyvaultKeyStatus.Reset()
	m.prometheus.keyvaultSecretInfo.Reset()
	m.prometheus.keyvaultSecretStatus.Reset()
	m.prometheus.keyvaultCertificateInfo.Reset()
	m.prometheus.keyvaultCertificateStatus.Reset()
}

func (m *MetricsCollectorKeyvault) Collect(ctx context.Context, logger *log.Entry, callback chan<- func(), subscription subscriptions.Subscription) {
	var keyvaultResult keyvaultMgmt.VaultListResultIterator
	var err error
	var wg sync.WaitGroup

	keyvaultClient := keyvaultMgmt.NewVaultsClientWithBaseURI(azureEnvironment.KeyVaultEndpoint, *subscription.SubscriptionID)
	keyvaultClient.Authorizer = AzureAuthorizer

	if opts.Azure.ResourceGroup != "" {
		keyvaultResult, err = keyvaultClient.ListByResourceGroupComplete(ctx, opts.Azure.ResourceGroup, nil)
	} else {
		keyvaultResult, err = keyvaultClient.ListBySubscriptionComplete(ctx, nil)
	}

	if err != nil {
		panic(err)
	}

	keyvaultCount := 0
	for keyvaultResult.NotDone() {
		keyvaultItem := keyvaultResult.Value()

		contextLogger := logger.WithField("keyvault", to.String(keyvaultItem.Name))

		client := keyvault.New()
		client.Authorizer = m.keyvaultAuth

		contextLogger.Info("collecting keyvault metrics")

		wg.Add(1)
		go func(ctx context.Context, subscription subscriptions.Subscription, client keyvault.BaseClient, vault keyvaultMgmt.Vault) {
			defer wg.Done()
			m.collectKeyvault(ctx, contextLogger, callback, subscription, client, vault)
		}(ctx, subscription, client, keyvaultItem)

		if keyvaultResult.NextWithContext(ctx) != nil {
			break
		}

		keyvaultCount++
	}

	wg.Wait()
}

func (m *MetricsCollectorKeyvault) collectKeyvault(ctx context.Context, logger *log.Entry, callback chan<- func(), subscription subscriptions.Subscription, client keyvault.BaseClient, vault keyvaultMgmt.Vault) (status bool) {
	status = true

	vaultMetrics := prometheusCommon.NewMetricsList()
	vaultStatusMetrics := prometheusCommon.NewMetricsList()
	vaultKeyMetrics := prometheusCommon.NewMetricsList()
	vaultKeyStatusMetrics := prometheusCommon.NewMetricsList()
	vaultSecretMetrics := prometheusCommon.NewMetricsList()
	vaultSecretStatusMetrics := prometheusCommon.NewMetricsList()
	vaultCertificateMetrics := prometheusCommon.NewMetricsList()
	vaultCertificateStatusMetrics := prometheusCommon.NewMetricsList()

	vaultUrl := fmt.Sprintf("https://%s.%s", to.String(vault.Name), azureEnvironment.KeyVaultDNSSuffix)

	// ########################
	// Keyvault
	// ########################

	vaultLabels := prometheus.Labels{
		"subscriptionID": to.String(subscription.SubscriptionID),
		"resourceID":     to.String(vault.ID),
		"vaultName":      to.String(vault.Name),
		"location":       to.String(vault.Location),
		"resourceGroup":  extractResourceGroupFromAzureId(*vault.ID),
	}
	vaultLabels = azureKeyvaultTag.appendPrometheusLabel(vaultLabels, vault.Tags)
	vaultMetrics.AddInfo(vaultLabels)

	// ########################
	// Keys
	// ########################

	keyResult, err := client.GetKeysComplete(ctx, vaultUrl, nil)
	if err != nil {
		logger.Warning(err)
		vaultStatusMetrics.Add(prometheus.Labels{
			"subscriptionID": to.String(subscription.SubscriptionID),
			"resourceID":     to.String(vault.ID),
			"vaultName":      to.String(vault.Name),
			"type":           "access",
			"scope":          "keys",
		}, 0)
	} else {
		vaultStatusMetrics.Add(prometheus.Labels{
			"subscriptionID": to.String(subscription.SubscriptionID),
			"resourceID":     to.String(vault.ID),
			"vaultName":      to.String(vault.Name),
			"type":           "access",
			"scope":          "keys",
		}, 1)
	}

	for keyResult.NotDone() {
		item := keyResult.Value()

		vaultKeyMetrics.AddInfo(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"keyID":     to.String(item.Kid),
			"enabled":   boolToString(*item.Attributes.Enabled),
		})

		// expiry date
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"keyID":     to.String(item.Kid),
			"type":      "expiry",
		}, expiryDate)

		// not before
		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"keyID":     to.String(item.Kid),
			"type":      "notBefore",
		}, notBeforeDate)

		// created
		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"keyID":     to.String(item.Kid),
			"type":      "created",
		}, createdDate)

		// updated
		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"keyID":     to.String(item.Kid),
			"type":      "updated",
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
		logger.Warning(err)
		vaultStatusMetrics.Add(prometheus.Labels{
			"subscriptionID": to.String(subscription.SubscriptionID),
			"resourceID":     to.String(vault.ID),
			"vaultName":      to.String(vault.Name),
			"type":           "access",
			"scope":          "secrets",
		}, 0)
	} else {
		vaultStatusMetrics.Add(prometheus.Labels{
			"subscriptionID": to.String(subscription.SubscriptionID),
			"resourceID":     to.String(vault.ID),
			"vaultName":      to.String(vault.Name),
			"type":           "access",
			"scope":          "secrets",
		}, 1)
	}

	for secretsResult.NotDone() {
		item := secretsResult.Value()

		vaultSecretMetrics.AddInfo(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"secretID":  to.String(item.ID),
			"enabled":   boolToString(to.Bool(item.Attributes.Enabled)),
		})

		// expiry date
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"secretID":  to.String(item.ID),
			"type":      "expiry",
		}, expiryDate)

		// notbefore
		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"secretID":  to.String(item.ID),
			"type":      "notBefore",
		}, notBeforeDate)

		// created
		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"secretID":  to.String(item.ID),
			"type":      "created",
		}, createdDate)

		// updated
		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"vaultName": to.String(vault.Name),
			"secretID":  to.String(item.ID),
			"type":      "updated",
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
		logger.Warning(err)
		vaultStatusMetrics.Add(prometheus.Labels{
			"subscriptionID": to.String(subscription.SubscriptionID),
			"resourceID":     to.String(vault.ID),
			"vaultName":      to.String(vault.Name),
			"type":           "access",
			"scope":          "certificates",
		}, 0)
	} else {
		vaultStatusMetrics.Add(prometheus.Labels{
			"subscriptionID": to.String(subscription.SubscriptionID),
			"resourceID":     to.String(vault.ID),
			"vaultName":      to.String(vault.Name),
			"type":           "access",
			"scope":          "certificates",
		}, 1)
	}

	for certificateResult.NotDone() {
		item := certificateResult.Value()

		vaultCertificateMetrics.AddInfo(prometheus.Labels{
			"vaultName":     to.String(vault.Name),
			"certificateID": to.String(item.ID),
			"enabled":       boolToString(to.Bool(item.Attributes.Enabled)),
		})

		// expiry
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0, 0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"vaultName":     to.String(vault.Name),
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
			"vaultName":     to.String(vault.Name),
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
			"vaultName":     to.String(vault.Name),
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
			"vaultName":     to.String(vault.Name),
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
	}

	return
}
