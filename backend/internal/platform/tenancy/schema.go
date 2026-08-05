package tenancy

import (
	"fmt"
	"strings"
)

func GetSchemaName(tenantCode string) string {
	if tenantCode == "" || tenantCode == "public" {
		return "public"
	}
	code := strings.ToLower(tenantCode)
	if code == "pekan" || code == "pekanhonet" {
		return "wkspid_pekan_pekanhonet"
	}
	return fmt.Sprintf("wkspid_pekan_%s", code)
}
