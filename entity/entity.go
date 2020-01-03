package entity

// Message structure
type Message struct {
	UUID string `json:"uuid"`
	Path string `json:"path"`
	Data string `json:"data"`
}

// Output structure
type Output struct {
	Login string  `json:"login"`
	UUID  string  `json:"uuid"`
	Value Message `json:"value"`
	Time  string  `json:"time"`
}

// Heartbeat structure
type Heartbeat struct {
	Login string `json:"login"`
	UUID  string `json:"source"`
	Path  string `json:"path"`
	Data  string `json:"data"`
	Time  string `json:"time"`
}
