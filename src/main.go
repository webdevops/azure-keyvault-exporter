package main

import (
	"os"
	"fmt"
	"time"
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/jessevdk/go-flags"
)

const (
	Author  = "webdevops.io"
	Version = "0.1.0"
	AZURE_KEYVAULT_TAG_PREFIX = "tag_"

)

var (
	argparser          *flags.Parser
	args               []string
	Logger             *DaemonLogger
	ErrorLogger        *DaemonLogger
	AzureAuthorizer    autorest.Authorizer
	AzureSubscriptions []subscriptions.Subscription
)

var opts struct {
	// general settings
	Verbose     []bool `         long:"verbose" short:"v"         env:"VERBOSE"                             description:"Verbose mode"`

	// server settings
	ServerBind  string `         long:"bind"                      env:"SERVER_BIND"                         description:"Server address"                                   default:":8080"`
	ScrapeTime  time.Duration `  long:"scrape-time"               env:"SCRAPE_TIME"                         description:"Scrape time (time.duration)"                      default:"3h"`

	AzureSubscription  []string `long:"azure-subscription"        env:"AZURE_SUBSCRIPTION_ID"               description:"Azure Subscription ID"`
	AzureResourceGroup string `  long:"azure-resourcegroup"       env:"AZURE_RESOURCEGROUP"                 description:"Azure ResourceGroup"`
	AzureKeyvaultCount int `     long:"azure-keyvalut-count"      env:"AZURE_KEYVAULT_COUNT"                description:"Azure Keyvault count" default:"100"`
	AzureKeyvaultTag []string `  long:"azure-keyvault-tag"        env:"AZURE_KEYVAULT_TAG"   env-delim:" "  description:"Azure ResourceGroup tags"                         default:"owner"`

}

func main() {
	initArgparser()

	// Init logger
	Logger = CreateDaemonLogger(0)
	ErrorLogger = CreateDaemonErrorLogger(0)

	// set verbosity

	Verbose = len(opts.Verbose) >= 1
	Logger.Messsage("Init Azure Keyvault exporter v%s (written by %v)", Version, Author)

	Logger.Messsage("Init Azure connection")
	initAzureConnection()

	Logger.Messsage("Starting metrics collection")
	Logger.Messsage("  scape time: %v", opts.ScrapeTime)
	setupMetrics()
	startMetricsCollection()

	Logger.Messsage("Starting http server on %s", opts.ServerBind)
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
