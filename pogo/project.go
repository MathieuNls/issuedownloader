package pogo

// Comment represents a comment in a bug report
type Project interface {
	String() string
	Name() string
	SQLAuth(string)
	Url(string)
	GitPath(string)
	LastIngestedCommit() string
}
