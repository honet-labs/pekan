package domain

import "errors"

var (
	ErrInvalidFormat     = errors.New("invalid report format")
	ErrInvalidReportType = errors.New("invalid report type")
	ErrInvalidDateRange  = errors.New("invalid date range")
	ErrReportNotFound    = errors.New("report not found")
)

