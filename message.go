package grest

// DeleteMsg is delete result
type DeleteMsg struct {
	Count int `json:"count" description:"delete count"`
}

// NewDeleteMsg is create DeleteMsg
func NewDeleteMsg(count int) (msg *DeleteMsg) {
	return &DeleteMsg{Count: count}
}

// ErrorMsg is err message
type ErrorMsg struct {
	Error errorMsg `json:"error"`
}

// errorMsg is err messages
type errorMsg struct {
	StatusCode int         `json:"statusCode"`
	Name       string      `json:"name"`
	Message    interface{} `json:"message"`
}

// NewErrorMsg is create errorMsg
func NewErrorMsg(statusCode int, name string, message interface{}) (err *ErrorMsg) {
	return &ErrorMsg{Error: errorMsg{StatusCode: statusCode, Name: name, Message: message}}
}
