package tenancy

import "fmt"

func GetSchemaName(tenantCode string) string {
	if tenantCode == "" || tenantCode == "public" || tenantCode == "default" {
		return "public"
	}
	return fmt.Sprintf("wkspid_pekan_%s", tenantCode)
}
