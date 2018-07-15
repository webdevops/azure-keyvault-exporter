package main

import (
	"os"
	"fmt"
	"context"
	"github.com/jessevdk/go-flags"
	"azure-keyvault-exporter/src/logger"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
)

const (
	Author  = "webdevops.io"
	Version = "0.1.0"
)

var (
	argparser          *flags.Parser
	args               []string
	Logger             *logger.DaemonLogger
	ErrorLogger        *logger.DaemonLogger
	AzureAuthorizer    autorest.Authorizer
	AzureSubscriptions []subscriptions.Subscription
)

var opts struct {
	AzureSubscription  []string `   long:"azure-subscription"    env:"AZURE_SUBSCRIPTION_ID"   description:"Azure Subscription ID"`
	AzureResourceGroup string   `   long:"azure-resourcegroup"   env:"AZURE_RESOURCEGROUP"     description:"Azure ResourceGroup"`
	AzureKeyvaultCount int      `   long:"azure-keyvalut-count"  env:"AZURE_KEYVAULT_COUNT"    description:"Azure Keyvault count" default:"100"`
	ScrapeTime         int      `   long:"scrape-time"           env:"SCRAPE_TIME"             description:"Scrape time in seconds"        default:"360"`
}

func main() {
	initArgparser()

	// Init logger
	Logger = logger.CreateDaemonLogger(0)
	ErrorLogger = logger.CreateDaemonErrorLogger(0)

	Logger.Messsage("init azure connection")
	initAzureConnection()

	Logger.Messsage("starting metrics collection")
	initMetrics()

	Logger.Messsage("starting http server")
	startHttpServer()
}

func initArgparser() {
	argparser = flags.NewParser(&opts, flags.Default)
	_, err := argparser.Parse()

	// check if there is an parse error
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Println(err)
			fmt.Println()
			argparser.WriteHelp(os.Stdout)
			os.Exit(1)
		}
	}
}

func initAzureConnection() {
	var err error
	ctx := context.Background()

	// azure authorizer
	AzureAuthorizer, err = auth.NewAuthorizerFromEnvironment()
	if err != nil {
		panic(err)
	}
	subscriptionsClient := subscriptions.NewClient()
	subscriptionsClient.Authorizer = AzureAuthorizer

	if len(opts.AzureSubscription) == 0 {
		listResult, err := subscriptionsClient.List(ctx)
		if err != nil {
			panic(err)
		}
		AzureSubscriptions = listResult.Values()
	} else {
		AzureSubscriptions = []subscriptions.Subscription{}
		for _, subId := range opts.AzureSubscription {
			result, err := subscriptionsClient.Get(ctx, subId)
			if err != nil {
				panic(err)
			}
			AzureSubscriptions = append(AzureSubscriptions, result)
		}
	}

}
