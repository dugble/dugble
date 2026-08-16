package moolre

// delivery.go owns asynchronous reconciliation for Moolre. It is intentionally
// separate from Send: this is where SMS delivery-status and Sender ID approval-
// status checks belong. Moolre documents these as separate status operations.
