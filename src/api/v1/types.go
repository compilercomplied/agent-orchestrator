package v1

type TaskRequest struct {
	Task string `json:"task"`
}

type TaskResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	PodName string `json:"pod_name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
