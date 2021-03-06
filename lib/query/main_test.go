package query

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mithrandie/csvq/lib/cmd"
	"github.com/mithrandie/csvq/lib/value"
)

func GetTestFilePath(filename string) string {
	return filepath.Join(TestDir, filename)
}

var TestDir = filepath.Join(os.TempDir(), "csvq_query_test")
var TestDataDir string
var TestLocation = "America/Los_Angeles"
var NowForTest = time.Date(2012, 2, 3, 9, 18, 15, 0, GetTestLocation())

func GetTestLocation() *time.Location {
	l, _ := time.LoadLocation(TestLocation)
	return l
}

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	defer teardown()

	setup()
	return m.Run()
}

func setup() {
	if _, err := os.Stat(TestDir); err == nil {
		os.RemoveAll(TestDir)
	}

	flags := cmd.GetFlags()
	flags.Location = TestLocation
	flags.Now = "2012-02-03 09:18:15"

	wdir, _ := os.Getwd()
	TestDataDir = filepath.Join(wdir, "..", "..", "testdata", "csv")

	r, _ := os.Open(filepath.Join(TestDataDir, "empty.txt"))
	os.Stdin = r

	if _, err := os.Stat(TestDir); os.IsNotExist(err) {
		os.Mkdir(TestDir, 0755)
	}

	copyfile(filepath.Join(TestDir, "table_sjis.csv"), filepath.Join(TestDataDir, "table_sjis.csv"))
	copyfile(filepath.Join(TestDir, "table_noheader.csv"), filepath.Join(TestDataDir, "table_noheader.csv"))
	copyfile(filepath.Join(TestDir, "table_broken.csv"), filepath.Join(TestDataDir, "table_broken.csv"))
	copyfile(filepath.Join(TestDir, "table1.csv"), filepath.Join(TestDataDir, "table1.csv"))
	copyfile(filepath.Join(TestDir, "table1b.csv"), filepath.Join(TestDataDir, "table1b.csv"))
	copyfile(filepath.Join(TestDir, "table2.csv"), filepath.Join(TestDataDir, "table2.csv"))
	copyfile(filepath.Join(TestDir, "table4.csv"), filepath.Join(TestDataDir, "table4.csv"))
	copyfile(filepath.Join(TestDir, "group_table.csv"), filepath.Join(TestDataDir, "group_table.csv"))
	copyfile(filepath.Join(TestDir, "insert_query.csv"), filepath.Join(TestDataDir, "table1.csv"))
	copyfile(filepath.Join(TestDir, "update_query.csv"), filepath.Join(TestDataDir, "table1.csv"))
	copyfile(filepath.Join(TestDir, "delete_query.csv"), filepath.Join(TestDataDir, "table1.csv"))
	copyfile(filepath.Join(TestDir, "add_columns.csv"), filepath.Join(TestDataDir, "table1.csv"))
	copyfile(filepath.Join(TestDir, "drop_columns.csv"), filepath.Join(TestDataDir, "table1.csv"))
	copyfile(filepath.Join(TestDir, "rename_column.csv"), filepath.Join(TestDataDir, "table1.csv"))
	copyfile(filepath.Join(TestDir, "updated_file_1.csv"), filepath.Join(TestDataDir, "table1.csv"))

	copyfile(filepath.Join(TestDir, "source.sql"), filepath.Join(filepath.Join(wdir, "..", "..", "testdata"), "source.sql"))
	copyfile(filepath.Join(TestDir, "source_syntaxerror.sql"), filepath.Join(filepath.Join(wdir, "..", "..", "testdata"), "source_syntaxerror.sql"))
}

func teardown() {
	if _, err := os.Stat(TestDir); err == nil {
		os.RemoveAll(TestDir)
	}
}

func initFlag() {
	flags := cmd.GetFlags()
	flags.Delimiter = cmd.UNDEF
	flags.Encoding = cmd.UTF8
	flags.LineBreak = cmd.LF
	flags.Repository = "."
	flags.DatetimeFormat = ""
	flags.NoHeader = false
	flags.WithoutNull = false
	flags.Stats = false
}

func copyfile(dstfile string, srcfile string) error {
	src, err := os.Open(srcfile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstfile)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}

func GenerateBenchGroupedViewFilter() Filter {
	primaries := make([]value.Primary, 10000)
	for i := 0; i < 10000; i++ {
		primaries[i] = value.NewInteger(int64(i))
	}

	view := &View{
		Header: NewHeader("table1", []string{"c1"}),
		RecordSet: []Record{
			{
				NewGroupCell(primaries),
			},
		},
		isGrouped: true,
	}

	return Filter{
		Records: []FilterRecord{
			{View: view},
		},
	}
}

func GenerateBenchView(tableName string, records int) *View {
	view := &View{
		Header:    NewHeader(tableName, []string{"c1"}),
		RecordSet: make([]Record, records),
	}

	for i := 0; i < records; i++ {
		view.RecordSet[i] = NewRecord([]value.Primary{value.NewInteger(int64(i))})
	}

	return view
}
