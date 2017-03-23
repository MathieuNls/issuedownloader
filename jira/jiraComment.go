package jira

// Comment represents a bugzilla comment
type Comment struct {
	Commenter string `xml:"author,attr"`
	Date      string `xml:"created,attr"`
	Text      string `xml:",chardata"`
}
