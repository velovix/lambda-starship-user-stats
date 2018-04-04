package datatypes

const REPLCommandKind = "REPLCommand"

type REPLCommand struct {
	UID       string `json:"uid"`
	Timestamp int64  `json:"timestamp"`
	Command   string `json:"command"`
}

const EditorContentKind = "EditorContent"

type EditorContent struct {
	UID       string `json:"uid"`
	Timestamp int64  `json:"timestamp"`
	Content   string `json:"content"`
}

const ErrorInstanceKind = "Error"

type ErrorInstance struct {
	UID         string `json:"uid"`
	Timestamp   int64  `json:"timestamp"`
	Description string `json:"description"`
}
