package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	azureCommon "github.com/webdevops/go-common/azure"
	"github.com/webdevops/go-common/prometheus/azuretracing"
	"github.com/webdevops/go-common/prometheus/collector"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/webdevops/azure-keyvault-exporter/config"
)

const (
	Author    = "webdevops.io"
	UserAgent = "azure-keyvault-exporter/"
)

var (
	argparser *flags.Parser
	opts      config.Opts

	logger *zap.SugaredLogger

	AzureClient                *azureCommon.Client
	AzureSubscriptionsIterator *azureCommon.IteratorSubscriptions

	// Git version information
	gitCommit = "<unknown>"
	gitTag    = "<unknown>"
)

func main() {
	initArgparser()
	defer initLogger()()

	logger.Infof("starting azure-keyvault-exporter v%s (%s; %s; by %v)", gitTag, gitCommit, runtime.Version(), Author)
	logger.Info(string(opts.GetJson()))

	logger.Infof("init Azure connection")
	initAzureConnection()

	logger.Infof("starting metrics collection")
	initMetricCollector()

	logger.Infof("Starting http server on %s", opts.ServerBind)
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

func initLogger() func() {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Encoding = "console"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// dev mode
	if opts.Logger.DevelopmentMode {
		loggerConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		// debug level
		if opts.Logger.Debug {
			loggerConfig = zap.NewDevelopmentConfig()
			loggerConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
			loggerConfig.DisableStacktrace = false
		}

		// json log format
		if opts.Logger.Json {
			loggerConfig.Encoding = "json"
			loggerConfig.EncoderConfig.TimeKey = ""
		}
	}

	zapLogger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}

	logger = zapLogger.Sugar()
	return func() {
		if err := logger.Sync(); err != nil {
			panic(err)
		}
	}
}

func initAzureConnection() {
	var err error
	AzureClient, err = azureCommon.NewClientFromEnvironment(*opts.Azure.Environment)
	if err != nil {
		logger.Panic(err.Error())
	}

	AzureClient.SetUserAgent(UserAgent + gitTag)

	AzureSubscriptionsIterator = azureCommon.NewIteratorSubscriptions(AzureClient)

	if len(opts.Azure.Subscription) >= 0 {
		subscriptionsClient := subscriptions.NewClientWithBaseURI(AzureClient.Environment.ResourceManagerEndpoint)
		AzureClient.DecorateAzureAutorest(&subscriptionsClient.Client)

		subscriptionList := []subscriptions.Subscription{}
		for _, subscriptionID := range opts.Azure.Subscription {
			subscription, err := subscriptionsClient.Get(context.Background(), subscriptionID)
			if err != nil {
				panic(err)
			}
			subscriptionList = append(subscriptionList, subscription)
		}

		AzureSubscriptionsIterator.SetFixedSubscriptions(subscriptionList)
	}
}

func initMetricCollector() {
	var collectorName string

	collectorName = "Keyvault"
	if opts.Scrape.Time.Seconds() > 0 {
		c := collector.New(collectorName, &MetricsCollectorKeyvault{}, logger.Desugar())
		c.SetScapeTime(opts.Scrape.Time)
		c.SetConcurrency(opts.Scrape.Concurrency)
		if err := c.Start(); err != nil {
			logger.Panic(err.Error())
		}
	} else {
		logger.Infow(
			"collector disabled",
			zap.String("collector", collectorName),
		)
	}
}

// start and handle prometheus handler
func startHttpServer() {
	// healthz
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "Ok"); err != nil {
			logger.Error(err)
		}
	})

	http.Handle("/metrics", azuretracing.RegisterAzureMetricAutoClean(promhttp.Handler()))
	logger.Panic(http.ListenAndServe(opts.ServerBind, nil))
}
