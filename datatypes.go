package main

type replCommand struct {
	UID       string `json:"uid"`
	Timestamp int64  `json:"timestamp"`
	Command   string `json:"command"`
}

type editorContent struct {
	UID       string `json:"uid"`
	Timestamp int64  `json:"timestamp"`
	Content   string `json:"content"`
}

type errorInstance struct {
	UID         string `json:"uid"`
	Timestamp   int64  `json:"timestamp"`
	Description string `json:"description"`
}
