package pogo

//Change Represents a commit
type Change interface {
	NewCommit(parentHashes []string, commitHash string, authorName string,
		authorEmail string, authorDate string, authorDateUnixTimestamp string,
		message string, regexFix string, regexReviewer string, isP4 bool) *Change
	String() string
}
