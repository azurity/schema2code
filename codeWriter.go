package schema2code

import "io"

type CodeWriter struct {
	writer io.Writer
	tab    string
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
		w.writer.Write([]byte(w.tab))
	}
}

func (w *CodeWriter) endLine() {
	w.writer.Write([]byte("\n"))
}

func (w *CodeWriter) Write(data string) {
	w.writer.Write([]byte(data))
}
