// Code generated by "stringer -type cellState"; DO NOT EDIT.

package csv

import "fmt"

const _cellState_name = "cellStateBegincellStateInQuotecellStateInQuoteQuotecellStateInCellcellStateTrailingWhiteSpace"

var _cellState_index = [...]uint8{0, 14, 30, 51, 66, 93}

func (i cellState) String() string {
	if i >= cellState(len(_cellState_index)-1) {
		return fmt.Sprintf("cellState(%d)", i)
	}
	return _cellState_name[_cellState_index[i]:_cellState_index[i+1]]
}
