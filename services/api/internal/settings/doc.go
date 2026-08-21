// Package settings is a persistent key/value store for application
// configuration that outlives a process but is not part of any deployment:
// preferences, defaults, and whatever an operator has changed at runtime.
//
// Deploy-time configuration is a different thing and lives in
// internal/config. Env vars are read there, once, at boot. Settings are
// written by the running system and read back later.
//
// A value is a JSON [Document] addressed by a [Key]. The ports are
// deliberately byte-oriented; typed access is composed on top by the generic
// [Read] and [Write] helpers, so the interface an adapter implements stays
// three methods wide no matter how many types callers store through it.
//
// Two implementations ship: [FS], which keeps documents under a directory —
// .oma in the working directory by default — and [Memory], for tests. Both
// are held to the same guarantees by the suite in settingstest, because a
// seam nobody verifies is a claim rather than a fact (ADR-0002, ADR-0005).
package settings
