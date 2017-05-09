package bugzilla

// Comment represents a bugzilla comment
type Comment struct {
	Commenter string `xml:"who"`
	Order     int    `xml:"comment_count"`
	Date      string `xml:"bug_when"`
	Text      string `xml:"thetext"`
}
