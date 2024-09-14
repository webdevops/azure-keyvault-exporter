package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"

	flags "github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/webdevops/go-common/prometheus/collector"
	"go.uber.org/zap"

	"github.com/webdevops/go-common/azuresdk/armclient"
	"github.com/webdevops/go-common/azuresdk/prometheus/tracing"

	"github.com/webdevops/azure-keyvault-exporter/config"
)

const (
	Author    = "webdevops.io"
	UserAgent = "azure-keyvault-exporter/"
)

var (
	argparser *flags.Parser
	Opts      config.Opts

	AzureClient                *armclient.ArmClient
	AzureSubscriptionsIterator *armclient.SubscriptionsIterator
	AzureResourceTagManager    *armclient.ResourceTagManager

	// Git version information
	gitCommit = "<unknown>"
	gitTag    = "<unknown>"

	// cache config
	cacheTag = "v1"
)

func main() {
	initArgparser()
	initLogger()
	initSystem()

	logger.Infof("starting azure-keyvault-exporter v%s (%s; %s; by %v)", gitTag, gitCommit, runtime.Version(), Author)
	logger.Info(string(Opts.GetJson()))

	logger.Infof("init Azure connection")
	initAzureConnection()

	logger.Infof("starting metrics collection")
	initMetricCollector()

	logger.Infof("Starting http server on %s", Opts.Server.Bind)
	startHttpServer()
}

func initArgparser() {
	argparser = flags.NewParser(&Opts, flags.Default)
	_, err := argparser.Parse()

	// check if there is an parse error
	if err != nil {
		var flagsErr *flags.Error
		if ok := errors.As(err, &flagsErr); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Println()
			argparser.WriteHelp(os.Stdout)
			os.Exit(1)
		}
	}
}

func initAzureConnection() {
	var err error
	AzureClient, err = armclient.NewArmClientWithCloudName(*Opts.Azure.Environment, logger)
	if err != nil {
		logger.Fatal(err.Error())
	}

	AzureClient.SetUserAgent(UserAgent + gitTag)

	// limit subscriptions (if filter is set)
	if len(Opts.Azure.Subscription) >= 1 {
		AzureClient.SetSubscriptionID(Opts.Azure.Subscription...)
	}

	// init subscription iterator
	AzureSubscriptionsIterator = armclient.NewSubscriptionIterator(AzureClient)

	// init resource tag manager
	AzureResourceTagManager, err = AzureClient.TagManager.ParseTagConfig(Opts.Azure.ResourceTags)
	if err != nil {
		logger.Fatalf(`unable to parse resourceTag configuration "%s": %v"`, Opts.Azure.ResourceTags, err.Error())
	}
}

func initMetricCollector() {
	collectorName := "keyvault"
	if Opts.Scrape.Time.Seconds() > 0 {
		c := collector.New(collectorName, &MetricsCollectorKeyvault{}, logger)
		c.SetScapeTime(Opts.Scrape.Time)
		c.SetConcurrency(Opts.Scrape.Concurrency)
		c.SetCache(
			Opts.GetCachePath(collectorName+".json"),
			collector.BuildCacheTag(cacheTag, Opts.Azure, Opts.KeyVault),
		)
		if err := c.Start(); err != nil {
			logger.Fatal(err.Error())
		}
	} else {
		logger.With(zap.String("collector", collectorName)).Info("collector disabled")
	}
}

// start and handle prometheus handler
func startHttpServer() {
	mux := http.NewServeMux()

	// healthz
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "Ok"); err != nil {
			logger.Error(err)
		}
	})

	// readyz
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "Ok"); err != nil {
			logger.Error(err)
		}
	})

	mux.Handle("/metrics", tracing.RegisterAzureMetricAutoClean(promhttp.Handler()))

	srv := &http.Server{
		Addr:         Opts.Server.Bind,
		Handler:      mux,
		ReadTimeout:  Opts.Server.ReadTimeout,
		WriteTimeout: Opts.Server.WriteTimeout,
	}
	logger.Fatal(srv.ListenAndServe())
}
