package etrest

type DeleteMsg struct {
	Count int `json:"count" description:"delete count"`
}

func NewDeleteMsg(count int) (msg *DeleteMsg) {
	return &DeleteMsg{Count: count}
}

type ErrorMsg struct {
	Error errorMsg `json:"error"`
}

type errorMsg struct {
	StatusCode int         `json:"statusCode"`
	Name       string      `json:"name"`
	Message    interface{} `json:"message"`
}

func NewErrorMsg(statusCode int, name string, message interface{}) (err *ErrorMsg) {
	return &ErrorMsg{Error: errorMsg{StatusCode: statusCode, Name: name, Message: message}}
}
