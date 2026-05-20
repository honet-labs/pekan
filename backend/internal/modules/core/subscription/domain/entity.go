package domain

import "time"

type Subscription struct {
	ID                 string
	TenantID           string
	PlanID             string
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
}

type EffectiveEntitlements struct {
	Modules  []string
	Features []string
}

