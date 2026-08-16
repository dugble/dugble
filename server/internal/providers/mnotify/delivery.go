package mnotify

// delivery.go owns asynchronous reconciliation for mNotify. It is intentionally
// separate from Send: this is where SMS delivery-status and Sender ID approval-
// status checks belong once their provider responses are normalized.
