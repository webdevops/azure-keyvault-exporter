package main

import (
	"regexp"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
)

var (
	keyvaultObjectNameRegExp = regexp.MustCompile(`^https://[^/]+/[^/]+/([^/]+)(/?|/.*)`)
)

func stringPtrToStringLower(val *string) string {
	return strings.ToLower(to.String(val))
}
func stringToStringLower(val string) string {
	return strings.ToLower(val)
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
