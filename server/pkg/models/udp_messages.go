package models

// RegistrationMsg is the JSON packet a UDP client sends to register or
// unregister itself from the notification broadcast list.
type RegistrationMsg struct {
	Type   string `json:"type"`    // "register" | "unregister"
	UserID string `json:"user_id"`
}

// UDPAck is the JSON response the server sends back to a UDP client after
// processing a registration/unregistration packet.
type UDPAck struct {
	Status    string `json:"status"`  // "registered" | "unregistered" | "error"
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// Notification is the JSON payload the server broadcasts to every
// registered UDP client when a new manga chapter is released.
type Notification struct {
	Type      string `json:"type"`     // "new_chapter"
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}
