package peer

import (
	"encoding/json"
	"math/rand"
	"time"
)

type RespondFunc func(data any)

type AcceptFunc func(data json.RawMessage)

type RejectFunc func(errorCode int, errorReason string)

type PeerMsg struct {
	Request      bool `json:"request"`
	Response     bool `json:"response"`
	Ok           bool `json:"ok"`
	Notification bool `json:"notification"`
}

type Request struct {
	Request bool            `json:"request"`
	Id      int             `json:"id"`
	Method  string          `json:"method"`
	Data    json.RawMessage `json:"data"`
}

type Response struct {
	Response bool            `json:"response"`
	Id       int             `json:"id"`
	Ok       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
}

type ResponseError struct {
	Response    bool   `json:"response"`
	Id          int    `json:"id"`
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"errorCode"`
	ErrorReason string `json:"errorReason"`
}

type Notification struct {
	Notification bool            `json:"notification"`
	Method       string          `json:"method"`
	Data         json.RawMessage `json:"data"`
}

func RandInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return rand.Intn(max-min) + min
}

func GenerateRandomNumber() int {
	return RandInt(1000000, 9999999)
}
