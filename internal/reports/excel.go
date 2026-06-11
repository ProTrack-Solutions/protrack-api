package reports

import (
	"io"

	"github.com/xuri/excelize/v2"
)

func GenerateExcel(w io.Writer, sheetName string, headers []string, rows [][]any) error {
	f := excelize.NewFile()
	defer f.Close()

	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	f.SetActiveSheet(index)
	if sheetName != "Sheet1" {
		f.DeleteSheet("Sheet1")
	}

	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4F81BD"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	for rowIndex, rowData := range rows {
		for colIndex, value := range rowData {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			if err != nil {
				return err
			}
			f.SetCellValue(sheetName, cell, value)
		}
	}

	for i := range headers {
		colName, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}

		f.SetColWidth(sheetName, colName, colName, 20)
	}

	return f.Write(w)
}

func AddRow(rows [][]any, label string, colIndices ...int) [][]any {
	if len(rows) == 0 {
		return rows
	}

	numCols := len(rows[0])
	totalRow := make([]any, numCols)

	targetCol := numCols - 1

	sums := make(map[int]float64)
	for _, idx := range colIndices {
		sums[idx] = 0
	}

	for _, row := range rows {
		for _, idx := range colIndices {
			if idx < len(row) {
				if val, ok := row[idx].(float64); ok {
					sums[idx] += val
				} else if val, ok := row[idx].(int); ok {
					sums[idx] += float64(val)
				}
			}
		}
	}

	for idx, sum := range sums {
		if idx < len(totalRow) {
			totalRow[targetCol] = sum
		}
	}

	totalRow[targetCol-1] = label

	return append(rows, totalRow)
}

// func AddRow(rows [][]any, label string, colIndices ...int) [][]any {
// 	if len(rows) == 0 {
// 		return rows
// 	}

// 	numCols := len(rows[0])
// 	totalRow := make([]any, numCols)

// 	targetCol := numCols - 1

// 	var sum float64

// 	for _, row := range rows {
// 		if targetCol < len(row) {
// 			switch v := row[targetCol].(type) {
// 			case float64:
// 				sum += v
// 			case int32:
// 				sum += float64(v)
// 			case int:
// 				sum += float64(v)
// 			}
// 		}
// 	}

// 	totalRow[targetCol] = sum
// 	totalRow[targetCol-1] = label

// 	return append(rows, totalRow)
// }
