package domain

import (
	"reflect"
	"time"

	"github.com/google/uuid"
)

type ReportType string

const (
	ReportSales          ReportType = "sales"
	ReportPayments       ReportType = "payments"
	ReportInventory      ReportType = "products"
	ReportProfitProduct  ReportType = "profit_product"
	ReportProfitCategory ReportType = "profit_category"
)

type ReportRequest struct {
	Type      ReportType `json:"type" form:"type"`
	StartDate time.Time  `json:"start_date" form:"start_date" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate   time.Time  `json:"end_date" form:"end_date" time_format:"2006-01-02T15:04:05Z07:00"`
	Format    string     `json:"format" form:"format"`
}

type ReportResponse struct {
	Headers  []string `json:"headers"`
	Rows     [][]any  `json:"rows"`
	FileName string   `json:"file_name"`
}

func MapStructToReport(data interface{}) ([]string, [][]any) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil, nil
	}

	typ := v.Index(0).Type()
	var headers []string
	var fieldIndices []int

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("excel")
		if tag != "" {
			headers = append(headers, tag)
			fieldIndices = append(fieldIndices, i)
		}
	}

	var rows [][]any
	for i := 0; i < v.Len(); i++ {
		var row []any
		item := v.Index(i)
		for _, idx := range fieldIndices {
			val := item.Field(idx).Interface()

			switch v := val.(type) {
			case time.Time:
				val = v.Format("02/01/2006 15:04")
			case uuid.UUID:
				val = v.String()
			}

			row = append(row, val)
		}
		rows = append(rows, row)
	}

	return headers, rows
}
