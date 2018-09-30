package main

import (
	"log"
	"fmt"
	"math/rand"
	"sync"
	"time"
	"regexp"
	"context"
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	keyvaultMgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
)

var (
	resourceGroupFromResourceIdRegExp = regexp.MustCompile("/resourceGroups/([^/]*)")

	prometheusKeyvault *prometheus.GaugeVec
	prometheusKeyvaultKeyInfo *prometheus.GaugeVec
	prometheusKeyvaultKeyStatus *prometheus.GaugeVec
	prometheusKeyvaultSecretInfo *prometheus.GaugeVec
	prometheusKeyvaultSecretStatus *prometheus.GaugeVec
	prometheusKeyvaultCertificateInfo *prometheus.GaugeVec
	prometheusKeyvaultCertificateStatus *prometheus.GaugeVec
)


func setupMetrics() {
	prometheusKeyvault = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_info",
			Help: "Azure KeyVault informations",
		},
		append(
			[]string{"subscriptionID", "vaultName", "location", "resourceGroup"},
			prefixSlice(AZURE_KEYVAULT_TAG_PREFIX, opts.AzureKeyvaultTag)...
		),
	)

	// ------------------------------------------
	// key
	prometheusKeyvaultKeyInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_key_info",
			Help: "Azure KeyVault key informations",
		},
		[]string{"vaultName", "keyID"},
	)

	prometheusKeyvaultKeyStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_key_status",
			Help: "Azure KeyVault key status",
		},
		[]string{"keyID", "type"},
	)

	// ------------------------------------------
	// secret
	prometheusKeyvaultSecretInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_secret_info",
			Help: "Azure KeyVault secret informations",
		},
		[]string{"vaultName", "secretID"},
	)

	prometheusKeyvaultSecretStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_secret_status",
			Help: "Azure KeyVault secret status",
		},
		[]string{"secretID", "type"},
	)

	// ------------------------------------------
	// certificate
	prometheusKeyvaultCertificateInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_certificate_info",
			Help: "Azure KeyVault certificate informations",
		},
		[]string{"vaultName", "secretID"},
	)

	prometheusKeyvaultCertificateStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_keyvault_certificate_status",
			Help: "Azure KeyVault certificate status",
		},
		[]string{"certificateID", "type"},
	)


	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(prometheusKeyvault)
	prometheus.MustRegister(prometheusKeyvaultKeyInfo)
	prometheus.MustRegister(prometheusKeyvaultKeyStatus)
	prometheus.MustRegister(prometheusKeyvaultSecretInfo)
	prometheus.MustRegister(prometheusKeyvaultSecretStatus)
	prometheus.MustRegister(prometheusKeyvaultCertificateInfo)
	prometheus.MustRegister(prometheusKeyvaultCertificateStatus)
}

func startMetricsCollection() {
	rand.Seed(time.Now().UnixNano())

	go func() {
		for {
			probeCollect()

			sleepDuration := randomTime(opts.ScrapeTime, opts.ScrapeTimeRand)
			Logger.Messsage("run: sleeping %v", sleepDuration.String())
			time.Sleep(sleepDuration)
		}
	}()
}

func startHttpServer() {
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(opts.ServerBind, nil))
}

func probeCollect() {
	var err error
	var keyvaultResult keyvaultMgmt.VaultListResultIterator
	var wg sync.WaitGroup
	ctx := context.Background()

	Logger.Messsage("run: start")

	callbackChannel := make(chan func())

	keyvaultAuth, err := auth.NewAuthorizerFromEnvironmentWithResource("https://vault.azure.net")
	if err != nil {
		panic(err)
	}

	for _, subscription := range AzureSubscriptions {
		Logger.Messsage(
			"subscription[%v]: Starting metrics update",
			*subscription.SubscriptionID,
		)

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
			keyvaultUrl := fmt.Sprintf("https://%s.vault.azure.net", *keyvaultItem.Name)

			client := keyvault.New()
			client.Authorizer = keyvaultAuth

			Logger.Verbose("keyvault[%v]: Collecting keyvault metrics", *keyvaultItem.Name)

			wg.Add(1)
			go func(ctx context.Context, subscription subscriptions.Subscription, client keyvault.BaseClient, vault keyvaultMgmt.Vault, vaultUrl string) {
				defer wg.Done()

				status := collectKeyvault(ctx, subscription, client, vault, vaultUrl, callbackChannel)

				// collect resourceGroup
				resourceGroup := ""
				rgSubMatch := resourceGroupFromResourceIdRegExp.FindStringSubmatch(*keyvaultItem.ID)

				if len(rgSubMatch) >= 1 {
					resourceGroup = rgSubMatch[1]
				}

				// set labels
				rgLabels := prometheus.Labels{
					"subscriptionID": *subscription.SubscriptionID,
					"vaultName": *keyvaultItem.Name,
					"location": *keyvaultItem.Location,
					"resourceGroup": resourceGroup,
				}

				// add tags
				for _, rgTag := range opts.AzureKeyvaultTag {
					rgTabLabel := AZURE_KEYVAULT_TAG_PREFIX + rgTag

					if _, ok := keyvaultItem.Tags[rgTag]; ok {
						rgLabels[rgTabLabel] = *keyvaultItem.Tags[rgTag]
					} else {
						rgLabels[rgTabLabel] = ""
					}
				}

				callbackChannel <- func() {
					prometheusKeyvault.With(rgLabels).Set(boolToFloat64(status))
				}

			}(ctx, subscription, client, keyvaultItem, keyvaultUrl)

			if keyvaultResult.Next() != nil {
				break
			}

			keyvaultCount++
		}

		Logger.Messsage(
			"subscription[%v]: found %v KeyVaults",
			*subscription.SubscriptionID,
			keyvaultCount,
		)

	}

	go func() {
		var callbackList []func()
		for callback := range callbackChannel {
			callbackList = append(callbackList, callback)
		}

		prometheusKeyvault.Reset()
		prometheusKeyvaultKeyInfo.Reset()
		prometheusKeyvaultKeyStatus.Reset()
		prometheusKeyvaultSecretInfo.Reset()
		prometheusKeyvaultSecretStatus.Reset()
		prometheusKeyvaultCertificateInfo.Reset()
		prometheusKeyvaultCertificateStatus.Reset()
		for _, callback := range callbackList {
			callback()
		}

		Logger.Messsage("run: finished")
	}()

	wg.Wait()
	close(callbackChannel)
}

func collectKeyvault(ctx context.Context, subscription subscriptions.Subscription, client keyvault.BaseClient, vault keyvaultMgmt.Vault, vaultUrl string, callback chan<- func()) (status bool) {
	status = true

	// ########################
	// Keys
	// ########################

	keyResult, err := client.GetKeysComplete(ctx, vaultUrl, nil)
	if err != nil {
		ErrorLogger.Verbose("keyvault[%v]: %v", *vault.Name, err)
		status = false
	}

	for keyResult.NotDone() {
		item := keyResult.Value()

		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}

		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}

		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}

		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}

		callback <- func() {
			prometheusKeyvaultKeyInfo.With(prometheus.Labels{"vaultName": *vault.Name, "keyID": *item.Kid}).Set(boolToFloat64(*item.Attributes.Enabled))
			prometheusKeyvaultKeyStatus.With(prometheus.Labels{"keyID": *item.Kid, "type": "expiry"}).Set(expiryDate)
			prometheusKeyvaultKeyStatus.With(prometheus.Labels{"keyID": *item.Kid, "type": "notBefore"}).Set(notBeforeDate)
			prometheusKeyvaultKeyStatus.With(prometheus.Labels{"keyID": *item.Kid, "type": "created"}).Set(createdDate)
			prometheusKeyvaultKeyStatus.With(prometheus.Labels{"keyID": *item.Kid, "type": "updated"}).Set(updatedDate)
		}

		if keyResult.Next() != nil {
			break
		}
	}

	// ########################
	// Secrets
	// ########################

	secretsResult, err := client.GetSecretsComplete(ctx, vaultUrl, nil)
	if err != nil {
		ErrorLogger.Verbose("keyvault[%v]: %v", *vault.Name, err)
		status = false
	}

	for secretsResult.NotDone() {
		item := secretsResult.Value()

		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}

		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}

		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}

		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}

		callback <- func() {
			prometheusKeyvaultSecretInfo.With(prometheus.Labels{"vaultName": *vault.Name, "secretID": *item.ID}).Set(boolToFloat64(*item.Attributes.Enabled))
			prometheusKeyvaultSecretStatus.With(prometheus.Labels{"secretID": *item.ID, "type": "expiry"}).Set(expiryDate)
			prometheusKeyvaultSecretStatus.With(prometheus.Labels{"secretID": *item.ID, "type": "notBefore"}).Set(notBeforeDate)
			prometheusKeyvaultSecretStatus.With(prometheus.Labels{"secretID": *item.ID, "type": "created"}).Set(createdDate)
			prometheusKeyvaultSecretStatus.With(prometheus.Labels{"secretID": *item.ID, "type": "updated"}).Set(updatedDate)
		}

		if secretsResult.Next() != nil {
			break
		}
	}

	// ########################
	// Certificate
	// ########################

	certificateResult, err := client.GetCertificatesComplete(ctx, vaultUrl, nil)
	if err != nil {
		ErrorLogger.Verbose("keyvault[%v]: %v", *vault.Name, err)
		status = false
	}

	for certificateResult.NotDone() {
		item := certificateResult.Value()

		expiryDate := float64(0)
		if item.Attributes.Expires != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Expires.Duration())
			expiryDate = float64(timestamp.Unix())
		}

		notBeforeDate := float64(0)
		if item.Attributes.NotBefore != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.NotBefore.Duration())
			notBeforeDate = float64(timestamp.Unix())
		}

		createdDate := float64(0)
		if item.Attributes.Created != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Created.Duration())
			createdDate = float64(timestamp.Unix())
		}

		updatedDate := float64(0)
		if item.Attributes.Updated != nil {
			timestamp := time.Unix(0,0).Add(item.Attributes.Updated.Duration())
			updatedDate = float64(timestamp.Unix())
		}

		callback <- func() {
			prometheusKeyvaultCertificateInfo.With(prometheus.Labels{"vaultName": *vault.Name, "certificateID": *item.ID}).Set(boolToFloat64(*item.Attributes.Enabled))
			prometheusKeyvaultCertificateStatus.With(prometheus.Labels{"certificateID": *item.ID, "type": "expiry"}).Set(expiryDate)
			prometheusKeyvaultCertificateStatus.With(prometheus.Labels{"certificateID": *item.ID, "type": "notBefore"}).Set(notBeforeDate)
			prometheusKeyvaultCertificateStatus.With(prometheus.Labels{"certificateID": *item.ID, "type": "created"}).Set(createdDate)
			prometheusKeyvaultCertificateStatus.With(prometheus.Labels{"certificateID": *item.ID, "type": "updated"}).Set(updatedDate)
		}

		if certificateResult.Next() != nil {
			break
		}
	}

	return
}

