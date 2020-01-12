package entity

import "encoding/json"

// The json format for exporter messages
// {
// 	"dtype": "data_metric",
// 	"action": "insert",
// 	"userid": "xxxx", // login id - from collector
// 	"source": "xxxx", // uuid
// 	"path": "xxxx",
// 	"time": "xxxx",
// 	"timestamp": "xxxx", // from collector
// 	"data": {
// 		"xxx": "xxxx"
// 	}
// }

// Message structure
type Message struct {
	Kind      string          `json:"dtype"`
	Action    string          `json:"action"`
	UserID    string          `json:"userid"`
	Source    string          `json:"source"`
	Path      string          `json:"path"`
	Time      string          `json:"time"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// The yaml format in consul
// fixed_columns: ["userid", "source", "path", "timestamp"]
// relations:
//   - dtype: "data_heartbeat"
//     table: "heartbeat"
//     columns: ["status"]
//     action: "insert"

// The json format
// {
//     "dtype": "data_heartbeat",
//     "action": "insert",
//     "userid": "xxxx",
//     "source": "xxxx",
//     "path": "xxxx",
//     "timestamp": "xxxx",
//     "data": {
//         "status": "OK"
//     }
// }

// HeartbeatData structure
type HeartbeatData struct {
	Status string `json:"status"`
}

// Heartbeat structure
type Heartbeat struct {
	Kind      string        `json:"dtype"`
	Action    string        `json:"action"`
	UserID    string        `json:"userid"`
	Source    string        `json:"source"`
	Path      string        `json:"path"`
	Data      HeartbeatData `json:"data"`
	Timestamp string        `json:"timestamp"`
}
