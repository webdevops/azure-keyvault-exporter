package main

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest/to"
	log "github.com/sirupsen/logrus"
)

type CollectorGeneral struct {
	CollectorBase
	Processor CollectorProcessorGeneralInterface
}

func (m *CollectorGeneral) Run(scrapeTime time.Duration) {
	m.SetScrapeTime(scrapeTime)

	m.Processor.Setup(m)
	go func() {
		for {
			go func() {
				m.Collect()
			}()
			m.sleepUntilNextCollection()
		}
	}()
}

func (m *CollectorGeneral) Collect() {
	var wg sync.WaitGroup
	var wgCallback sync.WaitGroup

	ctx := context.Background()

	callbackChannel := make(chan func())

	m.collectionStart()

	for _, subscription := range m.AzureSubscriptions {
		wg.Add(1)
		go func(ctx context.Context, callback chan<- func(), subscription subscriptions.Subscription) {
			defer wg.Done()
			contextLogger := m.logger.WithFields(log.Fields{
				"azureSubscription": to.String(subscription.SubscriptionID),
			})
			m.Processor.Collect(ctx, contextLogger, callbackChannel, subscription)
		}(ctx, callbackChannel, subscription)
	}

	// collect metrics (callbacks) and proceses them
	wgCallback.Add(1)
	go func() {
		defer wgCallback.Done()
		var callbackList []func()
		for callback := range callbackChannel {
			callbackList = append(callbackList, callback)
		}

		// reset metric values
		m.Processor.Reset()

		// process callbacks (set metrics)
		for _, callback := range callbackList {
			callback()
		}
	}()

	// wait for all funcs
	wg.Wait()
	close(callbackChannel)
	wgCallback.Wait()

	m.collectionFinish()
}
