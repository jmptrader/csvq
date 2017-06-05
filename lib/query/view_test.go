package query

import (
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/mithrandie/csvq/lib/cmd"
	"github.com/mithrandie/csvq/lib/parser"
)

var fileInfoTests = []struct {
	Name       string
	FilePath   string
	Repository string
	Delimiter  rune
	Result     *FileInfo
	Error      string
}{
	{
		Name:       "CSV",
		FilePath:   "table1",
		Repository: path.Join("..", "..", "testdata", "csv"),
		Delimiter:  cmd.UNDEF,
		Result: &FileInfo{
			Path:      "table1.csv",
			Delimiter: ',',
		},
	},
	{
		Name:       "TSV",
		FilePath:   "table3",
		Repository: path.Join("..", "..", "testdata", "csv"),
		Delimiter:  cmd.UNDEF,
		Result: &FileInfo{
			Path:      "table3.tsv",
			Delimiter: '\t',
		},
	},
	{
		Name:      "Not Exist Error",
		FilePath:  "notexist",
		Delimiter: cmd.UNDEF,
		Error:     "file notexist does not exist",
	},
	{
		Name:      "Directory Error",
		FilePath:  "/",
		Delimiter: cmd.UNDEF,
		Error:     "/ is a directory",
	},
}

func TestNewFileInfo(t *testing.T) {
	for _, v := range fileInfoTests {
		repo := v.Repository
		if 0 < len(repo) {
			dir, _ := os.Getwd()
			repo = path.Join(dir, repo)
		}

		fileInfo, err := NewFileInfo(v.FilePath, repo, v.Delimiter)
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
		if path.Base(fileInfo.Path) != v.Result.Path {
			t.Errorf("%s: filepath = %s, want %s", v.Name, path.Base(fileInfo.Path), v.Result.Path)
		}
		if fileInfo.Delimiter != v.Result.Delimiter {
			t.Errorf("%s: delimiter = %q, want %q", v.Name, fileInfo.Delimiter, v.Result.Delimiter)
		}
	}
}

var newViewTests = []struct {
	Name     string
	Encoding cmd.Encoding
	NoHeader bool
	From     parser.FromClause
	Stdin    string
	Filter   Filter
	Result   *View
	Error    string
}{
	{
		Name: "Dual View",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{Object: parser.Dual{}},
			},
		},
		Result: &View{
			Header: []HeaderField{{}},
			Records: []Record{
				{
					NewCell(parser.NewNull()),
				},
			},
		},
	},
	{
		Name: "Load File",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Identifier{Literal: "table1.csv"},
				},
			},
		},
		Result: &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			Records: []Record{
				NewRecord(0, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("str1"),
				}),
				NewRecord(1, []parser.Primary{
					parser.NewString("2"),
					parser.NewString("str2"),
				}),
				NewRecord(2, []parser.Primary{
					parser.NewString("3"),
					parser.NewString("str3"),
				}),
			},
			FileInfo: &FileInfo{
				Path:      "table1.csv",
				Delimiter: ',',
			},
		},
	},
	{
		Name:  "Load From Stdin",
		From:  parser.FromClause{},
		Stdin: "column1,column2\n1,\"str1\"",
		Result: &View{
			Header: NewHeader("stdin", []string{"column1", "column2"}),
			Records: []Record{
				NewRecord(0, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("str1"),
				}),
			},
			FileInfo: &FileInfo{
				Delimiter: ',',
			},
		},
	},
	{
		Name: "Stdin Empty Error",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Stdin{Stdin: "stdin"},
				},
			},
		},
		Error: "stdin is empty",
	},
	{
		Name: "Load File Error",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Identifier{Literal: "notexist"},
				},
			},
		},
		Error: "file notexist does not exist",
	},
	{
		Name:     "Load SJIS File",
		Encoding: cmd.SJIS,
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Identifier{Literal: "table_sjis"},
				},
			},
		},
		Result: &View{
			Header: NewHeader("table_sjis", []string{"column1", "column2"}),
			Records: []Record{
				NewRecord(0, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("日本語"),
				}),
				NewRecord(1, []parser.Primary{
					parser.NewString("2"),
					parser.NewString("str"),
				}),
			},
		},
	},
	{
		Name:     "Load No Header File",
		NoHeader: true,
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Identifier{Literal: "table_noheader"},
				},
			},
		},
		Result: &View{
			Header: NewHeader("table_noheader", []string{"c1", "c2"}),
			Records: []Record{
				NewRecord(0, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("str1"),
				}),
				NewRecord(1, []parser.Primary{
					parser.NewString("2"),
					parser.NewString("str2"),
				}),
			},
		},
	},
	{
		Name: "Load Multiple File",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Identifier{Literal: "table1"},
				},
				parser.Table{
					Object: parser.Identifier{Literal: "table2"},
				},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table2", Column: INTERNAL_ID_FIELD},
				{Reference: "table2", Column: "column3", FromTable: true},
				{Reference: "table2", Column: "column4", FromTable: true},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(0),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(0),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(0),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(1),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(0),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(2),
					parser.NewString("4"),
					parser.NewString("str44"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(0),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(1),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(2),
					parser.NewString("4"),
					parser.NewString("str44"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewInteger(0),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewInteger(1),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewInteger(2),
					parser.NewString("4"),
					parser.NewString("str44"),
				}),
			},
		},
	},
	{
		Name: "Cross Join",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
						},
						JoinType: parser.Token{Token: parser.CROSS, Literal: "cross"},
					},
				},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table2", Column: INTERNAL_ID_FIELD},
				{Reference: "table2", Column: "column3", FromTable: true},
				{Reference: "table2", Column: "column4", FromTable: true},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(0),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(0),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(0),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(1),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(0),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(2),
					parser.NewString("4"),
					parser.NewString("str44"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(0),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(1),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(2),
					parser.NewString("4"),
					parser.NewString("str44"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewInteger(0),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewInteger(1),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewInteger(2),
					parser.NewString("4"),
					parser.NewString("str44"),
				}),
			},
		},
	},
	{
		Name: "Inner Join",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
						},
						Condition: parser.JoinCondition{
							On: parser.Comparison{
								LHS:      parser.Identifier{Literal: "table1.column1"},
								RHS:      parser.Identifier{Literal: "table2.column3"},
								Operator: parser.Token{Token: parser.COMPARISON_OP, Literal: "="},
							},
						},
					},
				},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table2", Column: INTERNAL_ID_FIELD},
				{Reference: "table2", Column: "column3", FromTable: true},
				{Reference: "table2", Column: "column4", FromTable: true},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(0),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewInteger(1),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
			},
		},
	},
	{
		Name: "Outer Join",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
						},
						Direction: parser.Token{Token: parser.LEFT, Literal: "left"},
						Condition: parser.JoinCondition{
							On: parser.Comparison{
								LHS:      parser.Identifier{Literal: "table1.column1"},
								RHS:      parser.Identifier{Literal: "table2.column3"},
								Operator: parser.Token{Token: parser.COMPARISON_OP, Literal: "="},
							},
						},
					},
				},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table2", Column: INTERNAL_ID_FIELD},
				{Reference: "table2", Column: "column3", FromTable: true},
				{Reference: "table2", Column: "column4", FromTable: true},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(0),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewNull(),
					parser.NewNull(),
					parser.NewNull(),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(0),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewInteger(1),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
			},
		},
	},
	{
		Name: "Join Left Side Table File Not Exist Error",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "notexist"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
						},
						JoinType: parser.Token{Token: parser.CROSS, Literal: "cross"},
					},
				},
			},
		},
		Error: "file notexist does not exist",
	},
	{
		Name: "Join Right Side Table File Not Exist Error",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "notexist"},
						},
						JoinType: parser.Token{Token: parser.CROSS, Literal: "cross"},
					},
				},
			},
		},
		Error: "file notexist does not exist",
	},
	{
		Name: "Load Subquery",
		From: parser.FromClause{
			Tables: []parser.Expression{
				parser.Table{
					Object: parser.Subquery{
						Query: parser.SelectQuery{
							SelectClause: parser.SelectClause{
								Select: "select",
								Fields: []parser.Expression{
									parser.Field{Object: parser.Identifier{Literal: "column1"}},
									parser.Field{Object: parser.Identifier{Literal: "column2"}},
								},
							},
							FromClause: parser.FromClause{
								Tables: []parser.Expression{
									parser.Table{Object: parser.Identifier{Literal: "table1"}},
								},
							},
						},
					},
				},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{
					Reference: "table1",
					Column:    "column1",
					FromTable: true,
				},
				{
					Reference: "table1",
					Column:    "column2",
					FromTable: true,
				},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewString("1"),
					parser.NewString("str1"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewString("2"),
					parser.NewString("str2"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewString("3"),
					parser.NewString("str3"),
				}),
			},
		},
	},
}

func TestNewView(t *testing.T) {
	tf := cmd.GetFlags()
	dir, _ := os.Getwd()
	tf.Repository = path.Join(dir, "..", "..", "testdata", "csv")

	for _, v := range newViewTests {
		tf.Delimiter = cmd.UNDEF
		tf.NoHeader = v.NoHeader
		if v.Encoding != 0 {
			tf.Encoding = v.Encoding
		} else {
			tf.Encoding = cmd.UTF8
		}

		var oldStdin *os.File
		if 0 < len(v.Stdin) {
			oldStdin = os.Stdin
			r, w, _ := os.Pipe()
			w.WriteString(v.Stdin)
			w.Close()
			os.Stdin = r
		}

		result, err := NewView(v.From, v.Filter)

		if 0 < len(v.Stdin) {
			os.Stdin = oldStdin
		}

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

		if v.Result.FileInfo != nil {
			if path.Base(result.FileInfo.Path) != path.Base(v.Result.FileInfo.Path) {
				t.Errorf("%s: filepath = %q, want %q", v.Name, path.Base(result.FileInfo.Path), path.Base(v.Result.FileInfo.Path))
			}
			if result.FileInfo.Delimiter != v.Result.FileInfo.Delimiter {
				t.Errorf("%s: delimiter = %q, want %q", v.Name, result.FileInfo.Delimiter, v.Result.FileInfo.Delimiter)
			}
		}
		result.FileInfo = nil
		v.Result.FileInfo = nil
		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %s, want %s", v.Name, result, v.Result)
		}
	}
}

func TestNewViewFromGroupedRecord(t *testing.T) {
	fr := FilterRecord{
		View: &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			Records: []Record{
				{
					NewGroupCell([]parser.Primary{parser.NewInteger(1), parser.NewInteger(2), parser.NewInteger(3)}),
					NewGroupCell([]parser.Primary{parser.NewInteger(1), parser.NewInteger(2), parser.NewInteger(3)}),
					NewGroupCell([]parser.Primary{parser.NewString("str1"), parser.NewString("str2"), parser.NewString("str3")}),
				},
			},
		},
		RecordIndex: 0,
	}
	expect := &View{
		Header: NewHeader("table1", []string{"column1", "column2"}),
		Records: []Record{
			{NewCell(parser.NewInteger(1)), NewCell(parser.NewInteger(1)), NewCell(parser.NewString("str1"))},
			{NewCell(parser.NewInteger(2)), NewCell(parser.NewInteger(2)), NewCell(parser.NewString("str2"))},
			{NewCell(parser.NewInteger(3)), NewCell(parser.NewInteger(3)), NewCell(parser.NewString("str3"))},
		},
	}

	result := NewViewFromGroupedRecord(fr)
	if !reflect.DeepEqual(result, expect) {
		t.Errorf("result = %q, want %q", result, expect)
	}
}

var viewWhereTests = []struct {
	Name   string
	View   *View
	Where  parser.WhereClause
	Result []int
	Error  string
}{
	{
		Name: "Where",
		View: &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			Records: []Record{
				NewRecord(1, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("str1"),
				}),
				NewRecord(2, []parser.Primary{
					parser.NewString("2"),
					parser.NewString("str2"),
				}),
				NewRecord(3, []parser.Primary{
					parser.NewString("3"),
					parser.NewString("str3"),
				}),
			},
		},
		Where: parser.WhereClause{
			Filter: parser.Comparison{
				LHS:      parser.Identifier{Literal: "column1"},
				RHS:      parser.NewInteger(2),
				Operator: parser.Token{Token: parser.COMPARISON_OP, Literal: "="},
			},
		},
		Result: []int{1},
	},
	{
		Name: "Where Filter Error",
		View: &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			Records: []Record{
				NewRecord(1, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("str1"),
				}),
				NewRecord(2, []parser.Primary{
					parser.NewString("2"),
					parser.NewString("str2"),
				}),
				NewRecord(3, []parser.Primary{
					parser.NewString("3"),
					parser.NewString("str3"),
				}),
			},
		},
		Where: parser.WhereClause{
			Filter: parser.Comparison{
				LHS:      parser.Identifier{Literal: "notexist"},
				RHS:      parser.NewInteger(2),
				Operator: parser.Token{Token: parser.COMPARISON_OP, Literal: "="},
			},
		},
		Error: "identifier = notexist: field does not exist",
	},
}

func TestView_Where(t *testing.T) {
	for _, v := range viewWhereTests {
		err := v.View.Where(v.Where)
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
		if !reflect.DeepEqual(v.View.filteredIndices, v.Result) {
			t.Errorf("%s: result = %s, want %s", v.Name, v.View.filteredIndices, v.Result)
		}
	}
}

var viewGroupByTests = []struct {
	Name       string
	View       *View
	GroupBy    parser.GroupByClause
	Result     *View
	IsGrouped  bool
	GroupItems []string
	Error      string
}{
	{
		Name: "Group By",
		View: &View{
			Header: NewHeader("table1", []string{"column1", "column2", "column3"}),
			Records: []Record{
				NewRecord(1, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewString("group1"),
				}),
				NewRecord(2, []parser.Primary{
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewString("group2"),
				}),
				NewRecord(3, []parser.Primary{
					parser.NewString("3"),
					parser.NewString("str3"),
					parser.NewString("group1"),
				}),
				NewRecord(4, []parser.Primary{
					parser.NewString("4"),
					parser.NewString("str4"),
					parser.NewString("group2"),
				}),
			},
		},
		GroupBy: parser.GroupByClause{
			Items: []parser.Expression{
				parser.Identifier{Literal: "column3"},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{
					Reference: "table1",
					Column:    INTERNAL_ID_FIELD,
				},
				{
					Reference: "table1",
					Column:    "column1",
					FromTable: true,
				},
				{
					Reference: "table1",
					Column:    "column2",
					FromTable: true,
				},
				{
					Reference:  "table1",
					Column:     "column3",
					FromTable:  true,
					IsGroupKey: true,
				},
			},
			Records: []Record{
				{
					NewGroupCell([]parser.Primary{parser.NewInteger(1), parser.NewInteger(3)}),
					NewGroupCell([]parser.Primary{parser.NewString("1"), parser.NewString("3")}),
					NewGroupCell([]parser.Primary{parser.NewString("str1"), parser.NewString("str3")}),
					NewGroupCell([]parser.Primary{parser.NewString("group1"), parser.NewString("group1")}),
				},
				{
					NewGroupCell([]parser.Primary{parser.NewInteger(2), parser.NewInteger(4)}),
					NewGroupCell([]parser.Primary{parser.NewString("2"), parser.NewString("4")}),
					NewGroupCell([]parser.Primary{parser.NewString("str2"), parser.NewString("str4")}),
					NewGroupCell([]parser.Primary{parser.NewString("group2"), parser.NewString("group2")}),
				},
			},
			isGrouped: true,
		},
	},
}

func TestView_GroupBy(t *testing.T) {
	for _, v := range viewGroupByTests {
		err := v.View.GroupBy(v.GroupBy)
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
		if !reflect.DeepEqual(v.View, v.Result) {
			t.Errorf("%s: result = %s, want %s", v.Name, v.View, v.Result)
		}
	}
}

var viewHavingTests = []struct {
	Name   string
	View   *View
	Having parser.HavingClause
	Result []int
	Record []Record
	Error  string
}{
	{
		Name: "Having",
		View: &View{
			Header: []HeaderField{
				{
					Reference: "table1",
					Column:    INTERNAL_ID_FIELD,
				},
				{
					Reference: "table1",
					Column:    "column1",
					FromTable: true,
				},
				{
					Reference: "table1",
					Column:    "column2",
					FromTable: true,
				},
				{
					Reference:  "table1",
					Column:     "column3",
					FromTable:  true,
					IsGroupKey: true,
				},
			},
			isGrouped: true,
			Records: []Record{
				{
					NewGroupCell([]parser.Primary{parser.NewString("1"), parser.NewString("3")}),
					NewGroupCell([]parser.Primary{parser.NewString("1"), parser.NewString("3")}),
					NewGroupCell([]parser.Primary{parser.NewString("str1"), parser.NewString("str3")}),
					NewGroupCell([]parser.Primary{parser.NewString("group1"), parser.NewString("group1")}),
				},
				{
					NewGroupCell([]parser.Primary{parser.NewString("2"), parser.NewString("4")}),
					NewGroupCell([]parser.Primary{parser.NewString("2"), parser.NewString("4")}),
					NewGroupCell([]parser.Primary{parser.NewString("str2"), parser.NewString("str4")}),
					NewGroupCell([]parser.Primary{parser.NewString("group2"), parser.NewString("group2")}),
				},
			},
		},
		Having: parser.HavingClause{
			Filter: parser.Comparison{
				LHS: parser.Function{
					Name: "sum",
					Option: parser.Option{
						Args: []parser.Expression{parser.Identifier{Literal: "column1"}},
					},
				},
				RHS:      parser.NewInteger(5),
				Operator: parser.Token{Token: parser.COMPARISON_OP, Literal: ">"},
			},
		},
		Result: []int{1},
	},
	{
		Name: "Having Filter Error",
		View: &View{
			Header: []HeaderField{
				{
					Reference: "table1",
					Column:    INTERNAL_ID_FIELD,
				},
				{
					Reference: "table1",
					Column:    "column1",
					FromTable: true,
				},
				{
					Reference: "table1",
					Column:    "column2",
					FromTable: true,
				},
				{
					Reference:  "table1",
					Column:     "column3",
					FromTable:  true,
					IsGroupKey: true,
				},
			},
			isGrouped: true,
			Records: []Record{
				{
					NewGroupCell([]parser.Primary{parser.NewString("1"), parser.NewString("3")}),
					NewGroupCell([]parser.Primary{parser.NewString("1"), parser.NewString("3")}),
					NewGroupCell([]parser.Primary{parser.NewString("str1"), parser.NewString("str3")}),
					NewGroupCell([]parser.Primary{parser.NewString("group1"), parser.NewString("group1")}),
				},
				{
					NewGroupCell([]parser.Primary{parser.NewString("2"), parser.NewString("4")}),
					NewGroupCell([]parser.Primary{parser.NewString("2"), parser.NewString("4")}),
					NewGroupCell([]parser.Primary{parser.NewString("str2"), parser.NewString("str4")}),
					NewGroupCell([]parser.Primary{parser.NewString("group2"), parser.NewString("group2")}),
				},
			},
		},
		Having: parser.HavingClause{
			Filter: parser.Comparison{
				LHS: parser.Function{
					Name: "sum",
					Option: parser.Option{
						Args: []parser.Expression{parser.Identifier{Literal: "notexist"}},
					},
				},
				RHS:      parser.NewInteger(5),
				Operator: parser.Token{Token: parser.COMPARISON_OP, Literal: ">"},
			},
		},
		Error: "identifier = notexist: field does not exist",
	},
	{
		Name: "Having Not Grouped",
		View: &View{
			Header: NewHeader("table1", []string{"column1", "column2", "column3"}),
			Records: []Record{
				NewRecord(1, []parser.Primary{
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewString("group2"),
				}),
				NewRecord(2, []parser.Primary{
					parser.NewString("4"),
					parser.NewString("str4"),
					parser.NewString("group2"),
				}),
			},
		},
		Having: parser.HavingClause{
			Filter: parser.Comparison{
				LHS: parser.Function{
					Name: "sum",
					Option: parser.Option{
						Args: []parser.Expression{parser.Identifier{Literal: "column1"}},
					},
				},
				RHS:      parser.NewInteger(5),
				Operator: parser.Token{Token: parser.COMPARISON_OP, Literal: ">"},
			},
		},
		Result: []int{0},
		Record: []Record{
			{
				NewGroupCell([]parser.Primary{parser.NewInteger(1), parser.NewInteger(2)}),
				NewGroupCell([]parser.Primary{parser.NewString("2"), parser.NewString("4")}),
				NewGroupCell([]parser.Primary{parser.NewString("str2"), parser.NewString("str4")}),
				NewGroupCell([]parser.Primary{parser.NewString("group2"), parser.NewString("group2")}),
			},
		},
	},
}

func TestView_Having(t *testing.T) {
	for _, v := range viewHavingTests {
		err := v.View.Having(v.Having)
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
		if !reflect.DeepEqual(v.View.filteredIndices, v.Result) {
			t.Errorf("%s: result = %s, want %s", v.Name, v.View.filteredIndices, v.Result)
		}
		if v.Record != nil {
			if !reflect.DeepEqual(v.View.Records, v.Record) {
				t.Errorf("%s: result = %s, want %s", v.Name, v.View.Records, v.Record)
			}
		}
	}
}

var viewSelectTests = []struct {
	Name   string
	View   *View
	Select parser.SelectClause
	Result *View
	Error  string
}{
	{
		Name: "Select",
		View: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table2", Column: INTERNAL_ID_FIELD},
				{Reference: "table2", Column: "column3", FromTable: true},
				{Reference: "table2", Column: "column4", FromTable: true},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(3),
					parser.NewString("1"),
					parser.NewString("str44"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
			},
		},
		Select: parser.SelectClause{
			Fields: []parser.Expression{
				parser.Field{Object: parser.Identifier{Literal: "column2"}},
				parser.Field{Object: parser.AllColumns{}},
				parser.Field{Object: parser.NewInteger(1), Alias: parser.Identifier{Literal: "a"}},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table2", Column: INTERNAL_ID_FIELD},
				{Reference: "table2", Column: "column3", FromTable: true},
				{Reference: "table2", Column: "column4", FromTable: true},
				{Alias: "a"},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str22"),
					parser.NewInteger(1),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str33"),
					parser.NewInteger(1),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(3),
					parser.NewString("1"),
					parser.NewString("str44"),
					parser.NewInteger(1),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str22"),
					parser.NewInteger(1),
				}),
			},
			selectFields: []int{2, 1, 2, 4, 5, 6},
		},
	},
	{
		Name: "Select Distinct",
		View: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table2", Column: INTERNAL_ID_FIELD},
				{Reference: "table2", Column: "column3", FromTable: true},
				{Reference: "table2", Column: "column4", FromTable: true},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(2),
					parser.NewString("3"),
					parser.NewString("str33"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(1),
					parser.NewString("1"),
					parser.NewString("str1"),
					parser.NewInteger(3),
					parser.NewString("4"),
					parser.NewString("str44"),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewInteger(2),
					parser.NewString("2"),
					parser.NewString("str2"),
					parser.NewInteger(1),
					parser.NewString("2"),
					parser.NewString("str22"),
				}),
			},
		},
		Select: parser.SelectClause{
			Distinct: parser.Token{Token: parser.DISTINCT, Literal: "distinct"},
			Fields: []parser.Expression{
				parser.Field{Object: parser.Identifier{Literal: "column1"}},
				parser.Field{Object: parser.NewInteger(1), Alias: parser.Identifier{Literal: "a"}},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: "column1", FromTable: true},
				{Alias: "a"},
			},
			Records: []Record{
				NewRecordWithoutId([]parser.Primary{
					parser.NewString("1"),
					parser.NewInteger(1),
				}),
				NewRecordWithoutId([]parser.Primary{
					parser.NewString("2"),
					parser.NewInteger(1),
				}),
			},
			selectFields: []int{0, 1},
		},
	},
	{
		Name: "Select Aggregate Function",
		View: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
			},
			Records: []Record{
				NewRecord(1, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("str1"),
				}),
				NewRecord(2, []parser.Primary{
					parser.NewString("2"),
					parser.NewString("str2"),
				}),
			},
		},
		Select: parser.SelectClause{
			Fields: []parser.Expression{
				parser.Field{
					Object: parser.Function{
						Name: "sum",
						Option: parser.Option{
							Args: []parser.Expression{parser.Identifier{Literal: "column1"}},
						},
					},
				},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Alias: "sum(column1)"},
			},
			Records: []Record{
				{
					NewGroupCell([]parser.Primary{parser.NewInteger(1), parser.NewInteger(2)}),
					NewGroupCell([]parser.Primary{parser.NewString("1"), parser.NewString("2")}),
					NewGroupCell([]parser.Primary{parser.NewString("str1"), parser.NewString("str2")}),
					NewCell(parser.NewInteger(3)),
				},
			},
			selectFields: []int{3},
		},
	},
}

func TestView_Select(t *testing.T) {
	for _, v := range viewSelectTests {
		err := v.View.Select(v.Select)
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
		if !reflect.DeepEqual(v.View.Header, v.Result.Header) {
			t.Errorf("%s: header = %s, want %s", v.Name, v.View.Header, v.Result.Header)
		}
		if !reflect.DeepEqual(v.View.Records, v.Result.Records) {
			t.Errorf("%s: records = %s, want %s", v.Name, v.View.Records, v.Result.Records)
		}
		if !reflect.DeepEqual(v.View.selectFields, v.Result.selectFields) {
			t.Errorf("%s: select indices = %s, want %s", v.Name, v.View.selectFields, v.Result.selectFields)
		}
	}
}

var viewOrderByTests = []struct {
	Name    string
	View    *View
	OrderBy parser.OrderByClause
	Result  *View
	Error   string
}{
	{
		Name: "Order By",
		View: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table1", Column: "column3", FromTable: true},
			},
			Records: []Record{
				NewRecord(1, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("3"),
					parser.NewString("2"),
				}),
				NewRecord(2, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("4"),
					parser.NewString("3"),
				}),
				NewRecord(3, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("4"),
					parser.NewString("3"),
				}),
				NewRecord(4, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("3"),
					parser.NewNull(),
				}),
				NewRecord(5, []parser.Primary{
					parser.NewNull(),
					parser.NewString("2"),
					parser.NewString("4"),
				}),
			},
		},
		OrderBy: parser.OrderByClause{
			Items: []parser.Expression{
				parser.OrderItem{
					Item: parser.Identifier{Literal: "column1"},
				},
				parser.OrderItem{
					Item:      parser.Identifier{Literal: "column2"},
					Direction: parser.Token{Token: parser.DESC, Literal: "desc"},
				},
				parser.OrderItem{
					Item: parser.Identifier{Literal: "column3"},
				},
				parser.OrderItem{
					Item: parser.NewInteger(1),
				},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{Reference: "table1", Column: INTERNAL_ID_FIELD},
				{Reference: "table1", Column: "column1", FromTable: true},
				{Reference: "table1", Column: "column2", FromTable: true},
				{Reference: "table1", Column: "column3", FromTable: true},
				{Alias: "1"},
			},
			Records: []Record{
				NewRecord(2, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("4"),
					parser.NewString("3"),
					parser.NewInteger(1),
				}),
				NewRecord(3, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("4"),
					parser.NewString("3"),
					parser.NewInteger(1),
				}),
				NewRecord(4, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("3"),
					parser.NewNull(),
					parser.NewInteger(1),
				}),
				NewRecord(1, []parser.Primary{
					parser.NewString("1"),
					parser.NewString("3"),
					parser.NewString("2"),
					parser.NewInteger(1),
				}),
				NewRecord(5, []parser.Primary{
					parser.NewNull(),
					parser.NewString("2"),
					parser.NewString("4"),
					parser.NewInteger(1),
				}),
			},
		},
	},
}

func TestView_OrderBy(t *testing.T) {
	for _, v := range viewOrderByTests {
		err := v.View.OrderBy(v.OrderBy)
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
		if !reflect.DeepEqual(v.View.Header, v.Result.Header) {
			t.Errorf("%s: header = %s, want %s", v.Name, v.View.Header, v.Result.Header)
		}
		if !reflect.DeepEqual(v.View.Records, v.Result.Records) {
			t.Errorf("%s: records = %s, want %s", v.Name, v.View.Records, v.Result.Records)
		}
	}
}

func TestView_Limit(t *testing.T) {
	view := &View{
		Header: []HeaderField{
			{Reference: "table1", Column: INTERNAL_ID_FIELD},
			{Reference: "table1", Column: "column1", FromTable: true},
			{Reference: "table1", Column: "column2", FromTable: true},
		},
		Records: []Record{
			NewRecord(1, []parser.Primary{
				parser.NewString("1"),
				parser.NewString("str1"),
			}),
			NewRecord(2, []parser.Primary{
				parser.NewString("1"),
				parser.NewString("str1"),
			}),
			NewRecord(3, []parser.Primary{
				parser.NewString("1"),
				parser.NewString("str1"),
			}),
			NewRecord(4, []parser.Primary{
				parser.NewString("2"),
				parser.NewString("str2"),
			}),
		},
	}
	limit := parser.LimitClause{Number: 2}
	expect := &View{
		Header: []HeaderField{
			{Reference: "table1", Column: INTERNAL_ID_FIELD},
			{Reference: "table1", Column: "column1", FromTable: true},
			{Reference: "table1", Column: "column2", FromTable: true},
		},
		Records: []Record{
			NewRecord(1, []parser.Primary{
				parser.NewString("1"),
				parser.NewString("str1"),
			}),
			NewRecord(2, []parser.Primary{
				parser.NewString("1"),
				parser.NewString("str1"),
			}),
		},
	}

	view.Limit(limit)
	if !reflect.DeepEqual(view, expect) {
		t.Errorf("limit: view = %s, want %s", view, expect)
	}
}

func TestView_Fix(t *testing.T) {
	view := &View{
		Header: []HeaderField{
			{Reference: "table1", Column: INTERNAL_ID_FIELD},
			{Reference: "table1", Column: "column1", FromTable: true},
			{Reference: "table1", Column: "column2", FromTable: true},
		},
		Records: []Record{
			NewRecord(1, []parser.Primary{
				parser.NewString("1"),
				parser.NewString("str1"),
			}),
			NewRecord(2, []parser.Primary{
				parser.NewString("1"),
				parser.NewString("str1"),
			}),
		},
		selectFields: []int{2},
	}
	expect := &View{
		Header: []HeaderField{
			{Reference: "table1", Column: "column2", FromTable: true},
		},
		Records: []Record{
			NewRecordWithoutId([]parser.Primary{
				parser.NewString("str1"),
			}),
			NewRecordWithoutId([]parser.Primary{
				parser.NewString("str1"),
			}),
		},
		selectFields: []int(nil),
	}

	view.Fix()
	if !reflect.DeepEqual(view, expect) {
		t.Errorf("fix: view = %s, want %s", view, expect)
	}
}
