package tbbs

import (
	"fmt"
	strutil "github.com/pinpt/go-common/strings"
)

type RSTTableRow interface {
	Cols() []string
	FieldSize(col string, maxWdith int) (int, int)
	Data(col string, maxWidth int) []string
	Title(col string) string
	//FieldWidth(col int)
}

type RSTTable struct {
	Data []RSTTableRow
}

func (t RSTTable) Cols() []string {
	if len(t.Data) == 0 {
		return []string{}
	}
	return t.Data[0].Cols()
}

func (t RSTTable) NumLines() int {
	return len(t.Data)
}

func (t RSTTable) RowHeight(line int, widths map[string]int) int {
	if t.NumLines() <= line {
		return 0
	}
	rHeight := 0
	row := t.Data[line]
	for _, col := range row.Cols() {
		_, h := row.FieldSize(col, widths[col])
		if h > rHeight {
			rHeight = h
		}
	}
	return rHeight
}

func (t RSTTable) ColWidth(col string, maxWidth int) int {
	if t.NumLines() == 0 {
		return 0
	}
	width := 0
	for _, c := range t.Data {
		w, _ := c.FieldSize(col, maxWidth)
		if w > width {
			width = w
		}
	}
	tw := len(t.Data[0].Title(col))
	if tw > width {
		width = tw
	}
	return width + 2
}

func (t RSTTable) DrawTable() string {
	if t.NumLines() == 0 {
		return ""
	}
	row0 := t.Data[0]
	cols := row0.Cols()
	var maxWidth = 10 * 100 / len(cols)
	widths := make(map[string]int)
	for _, col := range cols {
		widths[col] = t.ColWidth(col, maxWidth)
	}
	tableLine := func(fill byte) string {
		s := "+"
		for _, col := range cols {
			//s += "+"
			//format := fmt.Sprintf("%% %s%dv+", fill, widths[col])
			//s += fmt.Sprintf(format, "")
			s += strutil.PadLeft("+", widths[col]+1, fill)
		}
		return s + "\n"
	}
	tableRow := func(line int) string {
		if line >= t.NumLines() {
			return ""
		}
		s := ""
		lines := t.RowHeight(line, widths)
		row := t.Data[line]
		for l := 0; l < lines; l++ {
			s += "|"
			for _, col := range cols {
				data := row.Data(col, widths[col])
				var txt string
				if l < len(data) {
					txt = data[l]
				}
				s += fmt.Sprintf(fmt.Sprintf("%% %ds|", widths[col]), txt)
			}
			s += "\n"
		}
		s += tableLine('-')
		return s
	}

	str := tableLine('-')
	str += "|"
	for _, col := range cols {
		str += fmt.Sprintf(fmt.Sprintf("%% %ds|", widths[col]), row0.Title(col))
	}
	str += "\n"
	str += tableLine('=')
	for r := 0; r < t.NumLines(); r++ {
		str += tableRow(r)
	}

	// build table header
	return str
}
