package main

import (
	"regexp"
)

var (
	resourceGroupFromResourceIdRegExp = regexp.MustCompile(`/subscriptions/[^/]+/resourceGroups/([^/]*)`)
	keyvaultObjectNameRegExp          = regexp.MustCompile(`^https://[^/]+/[^/]+/([^/]+)(/?|/.*)`)
)

func extractResourceGroupFromAzureId(azureId string) (resourceGroup string) {
	if subMatch := resourceGroupFromResourceIdRegExp.FindStringSubmatch(azureId); len(subMatch) >= 1 {
		resourceGroup = subMatch[1]
	}

	return
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func parseKeyvaultObjectName(objectId string) (objectName string) {
	if subMatch := keyvaultObjectNameRegExp.FindStringSubmatch(objectId); len(subMatch) >= 1 {
		objectName = subMatch[1]
	}

	return
}
