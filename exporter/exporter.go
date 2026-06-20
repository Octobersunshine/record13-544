package exporter

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"excel-export/task"

	"github.com/xuri/excelize/v2"
)

type ExportRequest struct {
	RowCount int `json:"row_count"`
}

type Exporter struct {
	tm          *task.Manager
	outputDir   string
}

func NewExporter(tm *task.Manager, outputDir string) (*Exporter, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	return &Exporter{
		tm:        tm,
		outputDir: outputDir,
	}, nil
}

func (e *Exporter) SubmitExport(req ExportRequest) (*task.Task, error) {
	if req.RowCount <= 0 {
		req.RowCount = 1000
	}
	if req.RowCount > 100000 {
		req.RowCount = 100000
	}

	t := e.tm.Create()

	go e.runExport(t.ID, req.RowCount)

	return t, nil
}

func (e *Exporter) runExport(taskID string, rowCount int) {
	defer func() {
		if r := recover(); r != nil {
			e.tm.SetFailed(taskID, fmt.Sprintf("panic: %v", r))
		}
	}()

	e.tm.SetRunning(taskID, rowCount)

	f := excelize.NewFile()
	sheetName := "Data"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		e.tm.SetFailed(taskID, fmt.Sprintf("create sheet: %v", err))
		return
	}
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)

	headers := []string{"序号", "姓名", "部门", "职位", "薪资", "入职日期", "邮箱", "电话", "地址", "状态"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	styleID, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
	})
	f.SetCellStyle(sheetName, "A1", "J1", styleID)

	departments := []string{"研发部", "产品部", "市场部", "销售部", "人力资源", "财务部", "运营部"}
	positions := []string{"工程师", "经理", "总监", "专员", "主管", "助理"}
	statuses := []string{"在职", "休假", "出差"}
	firstNames := []string{"张", "李", "王", "刘", "陈", "杨", "赵", "黄", "周", "吴", "徐", "孙", "马", "朱", "胡"}
	lastNames := []string{"伟", "芳", "娜", "敏", "静", "丽", "强", "磊", "军", "洋", "勇", "艳", "杰", "娟", "涛"}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	progressStep := rowCount / 100
	if progressStep < 1 {
		progressStep = 1
	}

	for i := 1; i <= rowCount; i++ {
		row := i + 1

		name := firstNames[r.Intn(len(firstNames))] + lastNames[r.Intn(len(lastNames))]
		dept := departments[r.Intn(len(departments))]
		pos := positions[r.Intn(len(positions))]
		salary := 8000 + r.Intn(42000)
		hireDate := time.Date(2015+r.Intn(10), time.Month(1+r.Intn(12)), 1+r.Intn(28), 0, 0, 0, 0, time.Local)
		email := fmt.Sprintf("user%d@example.com", 10000+i)
		phone := fmt.Sprintf("138%08d", r.Intn(100000000))
		address := fmt.Sprintf("北京市朝阳区%s街道%d号", departments[r.Intn(len(departments))], 1+r.Intn(999))
		status := statuses[r.Intn(len(statuses))]

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), name)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), dept)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), pos)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), salary)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), hireDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), email)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), phone)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), address)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), status)

		if i%progressStep == 0 || i == rowCount {
			e.tm.UpdateProgress(taskID, i)
			time.Sleep(10 * time.Millisecond)
		}
	}

	fileName := fmt.Sprintf("export_%s_%d.xlsx", taskID, time.Now().Unix())
	filePath := filepath.Join(e.outputDir, fileName)

	if err := f.SaveAs(filePath); err != nil {
		e.tm.SetFailed(taskID, fmt.Sprintf("save file: %v", err))
		return
	}

	e.tm.SetCompleted(taskID, fileName, filePath)
}
