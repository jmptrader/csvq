package action

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mithrandie/csvq/lib/cmd"
	"github.com/mithrandie/csvq/lib/query"
)

var showFieldsTests = []struct {
	Name  string
	Input string
	Error string
}{
	{
		Name:  "File Not Exist Error",
		Input: "notexist",
		Error: "file notexist does not exist",
	},
}

func TestShowFields(t *testing.T) {
	for _, v := range showFieldsTests {
		query.ReleaseResources()
		err := ShowFields(v.Input)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}
	}
	query.ReleaseResources()
}

func ExampleShowFields() {
	flags := cmd.GetFlags()
	dir, _ := os.Getwd()
	flags.Repository = filepath.Join(dir, "..", "..", "testdata", "csv")

	ShowFields("table1")
	//OUTPUT:
	//1. column1
	//2. column2
}
