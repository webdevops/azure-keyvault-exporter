package main

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/prometheus/client_golang/prometheus"
	keyvaultMgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"sync"
	"time"
)

type MetricsCollectorKeyvault struct {
	CollectorProcessorGeneral

	keyvaultAuth autorest.Authorizer

	prometheus struct {
		keyvault *prometheus.GaugeVec
		keyvaultKeyInfo *prometheus.GaugeVec
		keyvaultKeyStatus *prometheus.GaugeVec
		keyvaultSecretInfo *prometheus.GaugeVec
		keyvaultSecretStatus *prometheus.GaugeVec
		keyvaultCertificateInfo *prometheus.GaugeVec
		keyvaultCertificateStatus *prometheus.GaugeVec
	}
}

func (m *MetricsCollectorKeyvault) Setup(collector *CollectorGeneral) {
	var err error
	m.CollectorReference = collector

	m.keyvaultAuth, err = auth.NewAuthorizerFromEnvironmentWithResource("https://vault.azure.net")
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
				"accessible",
			},
			prefixSliceForPrometheusLabels(AZURE_KEYVAULT_TAG_PREFIX, opts.AzureKeyvaultTag)...
		),
	)

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


	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(m.prometheus.keyvault)
	prometheus.MustRegister(m.prometheus.keyvaultKeyInfo)
	prometheus.MustRegister(m.prometheus.keyvaultKeyStatus)
	prometheus.MustRegister(m.prometheus.keyvaultSecretInfo)
	prometheus.MustRegister(m.prometheus.keyvaultSecretStatus)
	prometheus.MustRegister(m.prometheus.keyvaultCertificateInfo)
	prometheus.MustRegister(m.prometheus.keyvaultCertificateStatus)
}

func (m *MetricsCollectorKeyvault) Reset() {
	m.prometheus.keyvault.Reset()
	m.prometheus.keyvaultKeyInfo.Reset()
	m.prometheus.keyvaultKeyStatus.Reset()
	m.prometheus.keyvaultSecretInfo.Reset()
	m.prometheus.keyvaultSecretStatus.Reset()
	m.prometheus.keyvaultCertificateInfo.Reset()
	m.prometheus.keyvaultCertificateStatus.Reset()
}

func (m *MetricsCollectorKeyvault) Collect(ctx context.Context, callback chan<- func(), subscription subscriptions.Subscription) {
	var keyvaultResult keyvaultMgmt.VaultListResultIterator
	var err error
	var wg sync.WaitGroup

	keyvaultClient := keyvaultMgmt.NewVaultsClient(*subscription.SubscriptionID)
	keyvaultClient.Authorizer = AzureAuthorizer

	if opts.AzureResourceGroup != "" {
		keyvaultResult, err = keyvaultClient.ListByResourceGroupComplete(ctx, opts.AzureResourceGroup, nil)
	} else {
		keyvaultResult, err = keyvaultClient.ListBySubscriptionComplete(ctx, nil)
	}

	if err != nil {
		panic(err)
	}

	keyvaultCount := 0
	for keyvaultResult.NotDone() {
		keyvaultItem := keyvaultResult.Value()

		client := keyvault.New()
		client.Authorizer = m.keyvaultAuth

		Logger.Infof("keyvault[%v]: Collecting keyvault metrics", *keyvaultItem.Name)

		wg.Add(1)
		go func(ctx context.Context, subscription subscriptions.Subscription, client keyvault.BaseClient, vault keyvaultMgmt.Vault) {
			defer wg.Done()
			m.collectKeyvault(ctx, callback, subscription, client, vault)
		}(ctx, subscription, client, keyvaultItem)

		if keyvaultResult.Next() != nil {
			break
		}

		keyvaultCount++
	}

	wg.Wait()
}

func (m *MetricsCollectorKeyvault) collectKeyvault(ctx context.Context, callback chan<- func(), subscription subscriptions.Subscription, client keyvault.BaseClient, vault keyvaultMgmt.Vault) (status bool) {
	status = true

	vaultMetrics := MetricCollectorList{}
	vaultKeyMetrics := MetricCollectorList{}
	vaultKeyStatusMetrics := MetricCollectorList{}
	vaultSecretMetrics := MetricCollectorList{}
	vaultSecretStatusMetrics := MetricCollectorList{}
	vaultCertificateMetrics := MetricCollectorList{}
	vaultCertificateStatusMetrics := MetricCollectorList{}


	vaultUrl := fmt.Sprintf("https://%s.vault.azure.net", *vault.Name)

	// ########################
	// Keys
	// ########################

	keyResult, err := client.GetKeysComplete(ctx, vaultUrl, nil)
	if err != nil {
		Logger.VerboseErrorf("keyvault[%v]: %v", *vault.Name, err)
		status = false
	}

	for keyResult.NotDone() {
		item := keyResult.Value()

		vaultKeyMetrics.AddInfo(prometheus.Labels{
			"vaultName": *vault.Name,
			"keyID": *item.Kid,
			"enabled": boolToString(*item.Attributes.Enabled),
		})

		// expiry date
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"keyID": *item.Kid,
			"type": "expiry",
		}, expiryDate)


		// not before
		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"keyID": *item.Kid,
			"type": "notBefore",
		}, notBeforeDate)

		// created
		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"keyID": *item.Kid,
			"type": "created",
		}, createdDate)


		// updated
		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}
		vaultKeyStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"keyID": *item.Kid,
			"type": "updated",
		}, updatedDate)

		if keyResult.Next() != nil {
			break
		}
	}

	// ########################
	// Secrets
	// ########################

	secretsResult, err := client.GetSecretsComplete(ctx, vaultUrl, nil)
	if err != nil {
		Logger.VerboseErrorf("keyvault[%v]: %v", *vault.Name, err)
		status = false
	}

	for secretsResult.NotDone() {
		item := secretsResult.Value()

		vaultSecretMetrics.AddInfo(prometheus.Labels{
			"vaultName": *vault.Name,
			"secretID": *item.ID,
			"enabled": boolToString(*item.Attributes.Enabled),
		})

		// expiry date
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"secretID": *item.ID,
			"type": "expiry",
		}, expiryDate)

		// notbefore
		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"secretID": *item.ID,
			"type": "notBefore",
		}, notBeforeDate)

		// created
		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"secretID": *item.ID,
			"type": "created",
		}, createdDate)

		// updated
		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}
		vaultSecretStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"secretID": *item.ID,
			"type": "updated",
		}, updatedDate)

		if secretsResult.Next() != nil {
			break
		}
	}

	// ########################
	// Certificate
	// ########################

	certificateResult, err := client.GetCertificatesComplete(ctx, vaultUrl, nil)
	if err != nil {
		Logger.VerboseErrorf("keyvault[%v]: %v", *vault.Name, err)
		status = false
	}

	for certificateResult.NotDone() {
		item := certificateResult.Value()

		vaultCertificateMetrics.AddInfo(prometheus.Labels{
			"vaultName": *vault.Name,
			"certificateID": *item.ID,
			"enabled": boolToString(*item.Attributes.Enabled),
		})

		// expiry
		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"certificateID": *item.ID,
			"type": "expiry",
		}, expiryDate)

		// notBefore
		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"certificateID": *item.ID,
			"type": "notBefore",
		}, notBeforeDate)

		// created
		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"certificateID": *item.ID,
			"type": "created",
		}, createdDate)

		// updated
		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}
		vaultCertificateStatusMetrics.Add(prometheus.Labels{
			"vaultName": *vault.Name,
			"certificateID": *item.ID,
			"type": "updated",
		}, updatedDate)

		if certificateResult.Next() != nil {
			break
		}
	}

	// ########################
	// Keyvault
	// ########################

	vaultLabels := prometheus.Labels{
		"subscriptionID": *subscription.SubscriptionID,
		"resourceID": *vault.ID,
		"vaultName": *vault.Name,
		"location": *vault.Location,
		"resourceGroup": extractResourceGroupFromAzureId(*vault.ID),
		"accessible": boolToString(status),
	}
	vaultLabels = addAzureResourceTags(vaultLabels, vault.Tags)
	vaultMetrics.AddInfo(vaultLabels)


	// ########################
	// Processing
	// ########################

	callback <- func() {
		vaultMetrics.GaugeSet(m.prometheus.keyvault)
		vaultKeyMetrics.GaugeSet(m.prometheus.keyvaultKeyInfo)
		vaultKeyStatusMetrics.GaugeSet(m.prometheus.keyvaultKeyStatus)
		vaultSecretMetrics.GaugeSet(m.prometheus.keyvaultSecretInfo)
		vaultSecretStatusMetrics.GaugeSet(m.prometheus.keyvaultSecretStatus)
		vaultCertificateMetrics.GaugeSet(m.prometheus.keyvaultCertificateInfo)
		vaultCertificateStatusMetrics.GaugeSet(m.prometheus.keyvaultCertificateStatus)
	}

	return
}
