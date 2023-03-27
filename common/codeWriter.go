package common

import "io"

type CodeWriter struct {
	Writer io.Writer
	Tab    string
	n      int
}

func (w *CodeWriter) Indent() {
	w.endLine()
	w.n += 1
	w.beginLine()
}

func (w *CodeWriter) Dedent() {
	w.endLine()
	w.n -= 1
	w.beginLine()
}

func (w *CodeWriter) CommonLine() {
	w.endLine()
	w.beginLine()
}

func (w *CodeWriter) beginLine() {
	for i := 0; i < w.n; i += 1 {
		w.Writer.Write([]byte(w.Tab))
	}
}

func (w *CodeWriter) endLine() {
	w.Writer.Write([]byte("\n"))
}

func (w *CodeWriter) Write(data string) {
	w.Writer.Write([]byte(data))
}
