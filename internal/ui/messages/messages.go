package messages

import "github.com/fragmede/nitpick/internal/api"

// View transition messages.
type (
	OpenStoryMsg  struct{ StoryID int }
	GoBackMsg     struct{}
	SwitchTabMsg  struct{ StoryType api.StoryType }
	OpenLoginMsg  struct{}
	OpenReplyMsg  struct{ ParentID int }
	OpenEditMsg   struct {
		ItemID      int
		CurrentText string
	}
	OpenUserMsg struct{ Username string }
	OpenNotifyMsg struct{}
	ShowHelpMsg   struct{}
)

// Data messages.
type (
	StoriesLoadedMsg struct {
		StoryType api.StoryType
		Items     []*api.Item
		Err       error
	}

	ItemLoadedMsg struct {
		Item *api.Item
		Err  error
	}

	CommentsLoadedMsg struct {
		StoryID int
		Items   []*api.Item
		Err     error
	}

	LoginResultMsg struct {
		Username string
		Err      error
	}

	ReplyResultMsg struct {
		ParentID int
		Err      error
	}

	EditResultMsg struct {
		ItemID int
		Err    error
	}

	SubmitResultMsg struct {
		Err error
	}

	OpenSubmitMsg struct{}

	VoteResultMsg struct {
		ItemID int
		Err    error
	}

	NewNotificationMsg struct {
		UnreadCount int
	}

	StatusMsg struct {
		Text    string
		IsError bool
	}

	SessionRestoredMsg struct {
		Username string
	}
)
