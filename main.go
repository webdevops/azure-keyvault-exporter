package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/go-common/prometheus/collector"

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
	opts      config.Opts

	AzureClient                *armclient.ArmClient
	AzureSubscriptionsIterator *armclient.SubscriptionsIterator

	// Git version information
	gitCommit = "<unknown>"
	gitTag    = "<unknown>"
)

func main() {
	initArgparser()
	initLogger()

	log.Infof("starting azure-keyvault-exporter v%s (%s; %s; by %v)", gitTag, gitCommit, runtime.Version(), Author)
	log.Info(string(opts.GetJson()))

	log.Infof("init Azure connection")
	initAzureConnection()

	log.Infof("starting metrics collection")
	initMetricCollector()

	log.Infof("Starting http server on %s", opts.ServerBind)
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
			fmt.Println()
			argparser.WriteHelp(os.Stdout)
			os.Exit(1)
		}
	}
}

func initLogger() {
	// verbose level
	if opts.Logger.Debug {
		log.SetLevel(log.DebugLevel)
	}

	// trace level
	if opts.Logger.Trace {
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, "/")
				funcName := s[len(s)-1]
				return funcName, fmt.Sprintf("%s:%d", f.File, f.Line)
			},
		})
	}

	// json log format
	if opts.Logger.Json {
		log.SetReportCaller(true)
		log.SetFormatter(&log.JSONFormatter{
			DisableTimestamp: true,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, "/")
				funcName := s[len(s)-1]
				return funcName, fmt.Sprintf("%s:%d", f.File, f.Line)
			},
		})
	}
}

func initAzureConnection() {
	var err error
	AzureClient, err = armclient.NewArmClientWithCloudName(*opts.Azure.Environment, log.StandardLogger())
	if err != nil {
		log.Panic(err.Error())
	}

	AzureClient.SetUserAgent(UserAgent + gitTag)

	// limit subscriptions (if filter is set)
	if len(opts.Azure.Subscription) >= 1 {
		AzureClient.SetSubscriptionFilter(opts.Azure.Subscription...)
	}

	AzureSubscriptionsIterator = armclient.NewSubscriptionIterator(AzureClient)
}

func initMetricCollector() {
	collectorName := "Keyvault"
	if opts.Scrape.Time.Seconds() > 0 {
		c := collector.New(collectorName, &MetricsCollectorKeyvault{}, log.StandardLogger())
		c.SetScapeTime(opts.Scrape.Time)
		c.SetConcurrency(opts.Scrape.Concurrency)
		if err := c.Start(); err != nil {
			log.Panic(err.Error())
		}
	} else {
		log.WithField("collector", collectorName).Info("collector disabled")
	}
}

// start and handle prometheus handler
func startHttpServer() {
	// healthz
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "Ok"); err != nil {
			log.Error(err)
		}
	})

	// readyz
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "Ok"); err != nil {
			log.Error(err)
		}
	})

	http.Handle("/metrics", tracing.RegisterAzureMetricAutoClean(promhttp.Handler()))
	log.Panic(http.ListenAndServe(opts.ServerBind, nil))
}
