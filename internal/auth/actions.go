package auth

// Actions are implemented directly on the Session type in session.go:
// - Session.Login()
// - Session.Reply()
// - Session.Vote()
//
// This keeps the auth flow cohesive since all actions require
// the same cookie-based session state.
