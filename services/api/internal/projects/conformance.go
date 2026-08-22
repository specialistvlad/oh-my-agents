package projects

// Registry implements the whole lifecycle. The assertion is here so a method
// drifting out of shape breaks the build rather than a call site.
var _ Store = (*Registry)(nil)
