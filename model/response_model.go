package model

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Code    int         `json:"code"`
	Status  int         `json:"status"`
}
