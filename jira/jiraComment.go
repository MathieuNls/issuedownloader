package jira

import (
	"github.com/mathieunls/deepchange-downloader/pogo"
)

// Comment represents a bugzilla comment
type Comment struct {
	pogo.CommentAttribut
}

func (c *Comment) String() string {
	return "\n{COMMENT={\n" +
		"\t {Commenter=" + c.Commenter + "}\n" +
		"\t {Date=" + c.Date + "}\n" +
		"\t {Text=" + c.Text + "}\n" +
		"}\n"
}
