package query

import (
	"sync"

	"github.com/mithrandie/csvq/lib/parser"

	"github.com/mithrandie/ternary"
)

func ParseJoinCondition(join parser.Join, view *View, joinView *View) (parser.QueryExpression, []parser.FieldReference, []parser.FieldReference, error) {
	if join.Natural.IsEmpty() && join.Condition == nil {
		return nil, nil, nil, nil
	}

	var using []parser.QueryExpression

	if !join.Natural.IsEmpty() {
		for _, field := range view.Header {
			if field.Column == INTERNAL_ID_COLUMN {
				continue
			}
			ref := parser.FieldReference{BaseExpr: parser.NewBaseExpr(join.Natural), Column: parser.Identifier{Literal: field.Column}}
			if _, err := joinView.FieldIndex(ref); err != nil {
				if _, ok := err.(*FieldAmbiguousError); ok {
					return nil, nil, nil, err
				}
				continue
			}
			using = append(using, parser.Identifier{BaseExpr: parser.NewBaseExpr(join.Natural), Literal: field.Column})
		}
	} else {
		cond := join.Condition.(parser.JoinCondition)
		if cond.On != nil {
			return cond.On, nil, nil, nil
		}

		using = cond.Using
	}

	if len(using) < 1 {
		return nil, nil, nil, nil
	}

	usingFields := make([]string, len(using))
	for i, v := range using {
		usingFields[i] = v.(parser.Identifier).Literal
	}

	includeFields := make([]parser.FieldReference, len(using))
	excludeFields := make([]parser.FieldReference, len(using))

	comps := make([]parser.Comparison, len(using))
	for i, v := range using {
		var lhs parser.FieldReference
		var rhs parser.FieldReference
		fieldref := parser.FieldReference{BaseExpr: v.GetBaseExpr(), Column: v.(parser.Identifier)}

		lhsidx, err := view.FieldIndex(fieldref)
		if err != nil {
			return nil, nil, nil, err
		}
		lhs = parser.FieldReference{BaseExpr: v.GetBaseExpr(), View: parser.Identifier{Literal: view.Header[lhsidx].View}, Column: v.(parser.Identifier)}

		rhsidx, err := joinView.FieldIndex(fieldref)
		if err != nil {
			return nil, nil, nil, err
		}
		rhs = parser.FieldReference{BaseExpr: v.GetBaseExpr(), View: parser.Identifier{Literal: joinView.Header[rhsidx].View}, Column: v.(parser.Identifier)}

		comps[i] = parser.Comparison{
			LHS:      lhs,
			RHS:      rhs,
			Operator: "=",
		}

		if join.Direction.Token == parser.RIGHT {
			includeFields[i] = rhs
			excludeFields[i] = lhs
		} else {
			includeFields[i] = lhs
			excludeFields[i] = rhs
		}
	}

	if len(comps) == 1 {
		return comps[0], includeFields, excludeFields, nil
	}

	logic := parser.Logic{
		LHS:      comps[0],
		RHS:      comps[1],
		Operator: parser.Token{Token: parser.AND, Literal: parser.TokenLiteral(parser.AND)},
	}
	for i := 2; i < len(comps); i++ {
		logic = parser.Logic{
			LHS:      logic,
			RHS:      comps[i],
			Operator: parser.Token{Token: parser.AND, Literal: parser.TokenLiteral(parser.AND)},
		}
	}
	return logic, includeFields, excludeFields, nil
}

func CrossJoin(view *View, joinView *View) {
	mergedHeader := MergeHeader(view.Header, joinView.Header)
	records := make(RecordSet, view.RecordLen()*joinView.RecordLen())

	cpu := NumberOfCPU(view.RecordLen())
	wg := sync.WaitGroup{}
	for i := 0; i < cpu; i++ {
		wg.Add(1)
		go func(thIdx int) {
			start, end := RecordRange(thIdx, view.RecordLen(), cpu)
			idx := start * joinView.RecordLen()

			for _, viewRecord := range view.RecordSet[start:end] {
				for _, joinViewRecord := range joinView.RecordSet {
					records[idx] = MergeRecord(viewRecord, joinViewRecord)
					idx++
				}
			}

			wg.Done()
		}(i)
	}
	wg.Wait()

	view.Header = mergedHeader
	view.RecordSet = records
	view.FileInfo = nil
}

func InnerJoin(view *View, joinView *View, condition parser.QueryExpression, parentFilter *Filter) error {
	if condition == nil {
		CrossJoin(view, joinView)
		return nil
	}

	mergedHeader := MergeHeader(view.Header, joinView.Header)

	var cpu int
	var splitLeft bool
	if joinView.RecordLen() < view.RecordLen() {
		cpu = NumberOfCPU(view.RecordLen())
		splitLeft = true
	} else {
		cpu = NumberOfCPU(joinView.RecordLen())
		splitLeft = false
	}

	var err error
	recordsList := make([]RecordSet, cpu)

	wg := sync.WaitGroup{}
	for i := 0; i < cpu; i++ {
		wg.Add(1)
		go func(thIdx int) {
			var lstart, lend, rstart, rend int
			var records RecordSet
			if splitLeft {
				lstart, lend = RecordRange(thIdx, view.RecordLen(), cpu)
				rstart = 0
				rend = joinView.RecordLen()
				records = make(RecordSet, 0, lend-lstart)
			} else {
				lstart = 0
				lend = view.RecordLen()
				rstart, rend = RecordRange(thIdx, joinView.RecordLen(), cpu)
				records = make(RecordSet, 0, rend-rstart)
			}

			filter := NewFilterForRecord(
				&View{
					Header:    mergedHeader,
					RecordSet: make(RecordSet, 1),
				},
				0,
				parentFilter,
			)

		InnerJoinLoop:
			for i := lstart; i < lend; i++ {
				for j := rstart; j < rend; j++ {
					if err != nil {
						break InnerJoinLoop
					}

					mergedRecord := MergeRecord(view.RecordSet[i], joinView.RecordSet[j])
					filter.Records[0].View.RecordSet[0] = mergedRecord

					primary, e := filter.Evaluate(condition)
					if e != nil {
						err = e
						break InnerJoinLoop
					}
					if primary.Ternary() == ternary.TRUE {
						records = append(records, mergedRecord)
					}
				}
			}

			recordsList[thIdx] = records
			wg.Done()
		}(i)
	}
	wg.Wait()

	if err != nil {
		return err
	}

	view.Header = mergedHeader
	view.RecordSet = MergeRecordSetList(recordsList)
	view.FileInfo = nil
	return nil
}

func OuterJoin(view *View, joinView *View, condition parser.QueryExpression, direction int, parentFilter *Filter) error {
	if direction == parser.TOKEN_UNDEFINED {
		direction = parser.LEFT
	}

	mergedHeader := MergeHeader(view.Header, joinView.Header)

	if direction == parser.RIGHT {
		view, joinView = joinView, view
	}

	viewEmptyRecord := NewEmptyRecord(view.FieldLen())
	joinViewEmptyRecord := NewEmptyRecord(joinView.FieldLen())

	var cpu int
	var splitLeft bool
	if joinView.RecordLen() < view.RecordLen() {
		cpu = NumberOfCPU(view.RecordLen())
		splitLeft = true
	} else {
		cpu = NumberOfCPU(joinView.RecordLen())
		splitLeft = false
	}

	var err error
	recordsList := make([]RecordSet, cpu)
	joinViewMatchesList := make([][]bool, cpu)

	wg := sync.WaitGroup{}
	for i := 0; i < cpu; i++ {
		wg.Add(1)
		go func(thIdx int) {
			var lstart, lend, rstart, rend int
			var records RecordSet
			if splitLeft {
				lstart, lend = RecordRange(thIdx, view.RecordLen(), cpu)
				rstart = 0
				rend = joinView.RecordLen()
				records = make(RecordSet, 0, lend-lstart)
			} else {
				lstart = 0
				lend = view.RecordLen()
				rstart, rend = RecordRange(thIdx, joinView.RecordLen(), cpu)
				records = make(RecordSet, 0, rend-rstart)
			}

			filter := NewFilterForRecord(
				&View{
					Header:    mergedHeader,
					RecordSet: make(RecordSet, 1),
				},
				0,
				parentFilter,
			)

			joinViewMatches := make([]bool, joinView.RecordLen())

		OuterJoinLoop:
			for i := lstart; i < lend; i++ {
				match := false
				for j := rstart; j < rend; j++ {
					if err != nil {
						break OuterJoinLoop
					}

					var mergedRecord Record
					switch direction {
					case parser.RIGHT:
						mergedRecord = MergeRecord(joinView.RecordSet[j], view.RecordSet[i])
					default:
						mergedRecord = MergeRecord(view.RecordSet[i], joinView.RecordSet[j])
					}
					filter.Records[0].View.RecordSet[0] = mergedRecord

					primary, e := filter.Evaluate(condition)
					if e != nil {
						err = e
						break OuterJoinLoop
					}
					if primary.Ternary() == ternary.TRUE {
						if direction == parser.FULL && !joinViewMatches[j] {
							joinViewMatches[j] = true
						}
						records = append(records, mergedRecord)
						match = true
					}
				}

				if !match {
					var record Record
					switch direction {
					case parser.RIGHT:
						record = MergeRecord(joinViewEmptyRecord, view.RecordSet[i])
					default:
						record = MergeRecord(view.RecordSet[i], joinViewEmptyRecord)
					}
					records = append(records, record)

				}
			}

			recordsList[thIdx] = records
			joinViewMatchesList[thIdx] = joinViewMatches
			wg.Done()
		}(i)
	}
	wg.Wait()

	if err != nil {
		return err
	}

	if direction == parser.FULL {
		for i := 0; i < joinView.RecordLen(); i++ {
			match := false
			for _, joinViewMatches := range joinViewMatchesList {
				if joinViewMatches[i] {
					match = true
					break
				}
			}
			if !match {
				record := MergeRecord(viewEmptyRecord, joinView.RecordSet[i])
				recordsList[len(recordsList)-1] = append(recordsList[len(recordsList)-1], record)
			}
		}
	}

	if direction == parser.RIGHT {
		view, joinView = joinView, view
	}

	view.Header = mergedHeader
	view.RecordSet = MergeRecordSetList(recordsList)
	view.FileInfo = nil
	return nil
}
