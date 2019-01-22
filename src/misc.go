package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"regexp"
	"time"
	"math/rand"
)

var (
	resourceGroupFromResourceIdRegExp = regexp.MustCompile("/resourceGroups/([^/]*)")
)

func extractResourceGroupFromAzureId (azureId string) (resourceGroup string) {
	if subMatch := resourceGroupFromResourceIdRegExp.FindStringSubmatch(azureId); len(subMatch) >= 1 {
		resourceGroup = subMatch[1]
	}

	return
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}


func prefixSlice(prefix string, valueMap []string) (ret []string) {
	for _, value := range valueMap {
		ret = append(ret, prefix + value)
	}
	return
}


func randomTime(base, randTime time.Duration) time.Duration {
	sleepTime := int(base.Seconds()) + rand.Intn(int(randTime.Seconds()))
	duration, err := time.ParseDuration(fmt.Sprintf("%ds", sleepTime))
	if err != nil {
		panic(err)
	}

	return duration
}

func timeToFloat64(v time.Time) float64 {
	return float64(v.Unix())
}

func addAzureResourceTags(labels prometheus.Labels, tags map[string]*string) (prometheus.Labels) {
	for _, rgTag := range opts.AzureKeyvaultTag {
		rgTabLabel := AZURE_KEYVAULT_TAG_PREFIX + rgTag

		if _, ok := tags[rgTag]; ok {
			labels[rgTabLabel] = *tags[rgTag]
		} else {
			labels[rgTabLabel] = ""
		}
	}

	return labels
}
