package main

import (
	"regexp"
)

var (
	resourceGroupFromResourceIdRegExp = regexp.MustCompile("/resourceGroups/([^/]*)")
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
