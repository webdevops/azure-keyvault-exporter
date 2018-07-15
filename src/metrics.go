package main

import (
	"time"
	"log"
	"fmt"
	"context"
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	keyvaultMgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
)

var (
	prometheusKeyvault = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_keyvault",
			Help: "Azure KeyVault",
		},
		[]string{"Subscription", "Keyvault"},
	)

	prometheusKeyvaultKeys = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_keyvault_key",
			Help: "Azure KeyVault key",
		},
		[]string{"Keyvault", "ID", "Expiry"},
	)

	prometheusKeyvaultSecrets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_keyvault_secret",
			Help: "Azure KeyVault secret",
		},
		[]string{"Keyvault", "ID", "Expiry"},
	)

	prometheusKeyvaultCertificates = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_keyvault_certificate",
			Help: "Azure KeyVault certificate",
		},
		[]string{"Keyvault", "ID", "Expiry"},
	)
)


func initMetrics() {
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(prometheusKeyvault)
	prometheus.MustRegister(prometheusKeyvaultKeys)
	prometheus.MustRegister(prometheusKeyvaultSecrets)
	prometheus.MustRegister(prometheusKeyvaultCertificates)

	go func() {
		for {
			probeCollect()
			time.Sleep(time.Duration(opts.ScrapeTime) * time.Second)
		}
	}()
}

func startHttpServer() {
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func probeCollect() {
	var err error
	var keyvaultResult keyvaultMgmt.VaultListResultIterator
	ctx := context.Background()

	keyvaultAuth, err := auth.NewAuthorizerFromEnvironmentWithResource("https://vault.azure.net")
	if err != nil {
		panic(err)
	}

	for _, subscription := range AzureSubscriptions {
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

		for keyvaultResult.NotDone() {
			keyvaultItem := keyvaultResult.Value()
			keyvaultUrl := fmt.Sprintf("https://%s.vault.azure.net", *keyvaultItem.Name)

			keyvaultItemClient := keyvault.New()
			keyvaultItemClient.Authorizer = keyvaultAuth

			prometheusKeyvault.With(prometheus.Labels{"Subscription": *subscription.ID, "Keyvault": *keyvaultItem.Name}).Set(1)

			// ########################
			// Keys
			// ########################

			keyResult, err := keyvaultItemClient.GetKeysComplete(ctx, keyvaultUrl, nil)
			if err != nil {
				Logger.Error(fmt.Sprintf("%s: failed to get keys", *keyvaultItem.Name), err)
				continue
			}

			for keyResult.NotDone() {
				keyItem := keyResult.Value()

				expiryString := ""
				expiryValue := float64(1)
				if keyItem.Attributes.NotBefore != nil {
					expiry := time.Unix(0,0).Add(keyItem.Attributes.Expires.Duration())
					expiryString = expiry.Format(time.RFC3339)
					expiryValue = float64(expiry.Unix())
				}

				prometheusKeyvaultKeys.With(prometheus.Labels{"Keyvault": *keyvaultItem.Name, "ID": *keyItem.Kid, "Expiry": expiryString}).Set(expiryValue)

				if keyResult.Next() != nil {
					break
				}
			}


			// ########################
			// Secrets
			// ########################

			secretsResult, err := keyvaultItemClient.GetSecretsComplete(ctx, keyvaultUrl, nil)
			if err != nil {
				Logger.Error(fmt.Sprintf("%s: failed to get secrets", *keyvaultItem.Name), err)
				continue
			}

			for secretsResult.NotDone() {
				secretItem := secretsResult.Value()

				expiryString := ""
				expiryValue := float64(1)
				if secretItem.Attributes.NotBefore != nil {
					expiry := time.Unix(0,0).Add(secretItem.Attributes.Expires.Duration())
					expiryString = expiry.Format(time.RFC3339)
					expiryValue = float64(expiry.Unix())
				}

				prometheusKeyvaultSecrets.With(prometheus.Labels{"Keyvault": *keyvaultItem.Name, "ID": *secretItem.ID, "Expiry": expiryString}).Set(expiryValue)

				if secretsResult.Next() != nil {
					break
				}
			}

			// ########################
			// Certificate
			// ########################

			certificateResult, err := keyvaultItemClient.GetCertificatesComplete(ctx, keyvaultUrl, nil)
			if err != nil {
				Logger.Error(fmt.Sprintf("%s: failed to get certificates", *keyvaultItem.Name), err)
				continue
			}

			for certificateResult.NotDone() {
				certificateItem := certificateResult.Value()

				expiryString := ""
				expiryValue := float64(1)
				if certificateItem.Attributes.NotBefore != nil {
					expiry := time.Unix(0,0).Add(certificateItem.Attributes.Expires.Duration())
					expiryString = expiry.Format(time.RFC3339)
					expiryValue = float64(expiry.Unix())
				}

				prometheusKeyvaultCertificates.With(prometheus.Labels{"Keyvault": *keyvaultItem.Name, "ID": *certificateItem.ID, "Expiry": expiryString}).Set(expiryValue)

				if certificateResult.Next() != nil {
					break
				}
			}

			if keyvaultResult.Next() != nil {
				break
			}
		}
	}
}

