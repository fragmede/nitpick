package monitor

// NewReplyNotification is sent when a new reply is detected.
type NewReplyNotification struct {
	ItemID      int
	ParentID    int
	StoryID     int
	ByUser      string
	TextPreview string
}
