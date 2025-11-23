package handler

type TaskRequest struct {
	Task string `json:"task"`
}

type TaskResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
