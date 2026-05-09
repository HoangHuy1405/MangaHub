package models

// ProgressUpdate is the JSON message a TCP client sends to the server
// when the user moves to a new chapter. The server broadcasts this to all
// other active connections belonging to the same user_id.
type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

// TCPResponse is the JSON acknowledgement the server sends back to the
// originating client after it processes a ProgressUpdate.
type TCPResponse struct {
	Status    string `json:"status"`  // "ok" | "error"
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}
