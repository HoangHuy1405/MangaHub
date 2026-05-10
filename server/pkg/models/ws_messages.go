package models

// MessageType is a typed string enum for WebSocket message routing.
// Using a dedicated type prevents typos and enables compile-time checks.
type MessageType string

const (
	MsgChat    MessageType = "chat"    // Regular user message
	MsgJoin    MessageType = "join"    // System notification when a user joins
	MsgLeave   MessageType = "leave"   // System notification when a user leaves
	MsgSystem  MessageType = "system"  // Generic server-side announcement
	MsgUsers   MessageType = "users"   // Server response listing online users
	MsgHistory MessageType = "history" // Server response with recent message history
	MsgError   MessageType = "error"   // Server-side error notification
	MsgPM      MessageType = "pm"      // Private message
)

// ChatMessage is the JSON envelope for all WebSocket chat messages.
// The Type field routes the message on both server and client side.
type ChatMessage struct {
	Type      MessageType `json:"type"`
	UserID    string      `json:"user_id"`
	Username  string      `json:"username"`
	Message   string      `json:"message"`
	Room      string      `json:"room,omitempty"`
	Target    string      `json:"target,omitempty"`
	Timestamp int64       `json:"timestamp"`
}
