package jira

import (
	"time"

	"github.com/BUMPER/IssueDownloader/pogo"
)

// JiraReport represents a bugzilla bug
type JiraReport struct {
	pogo.GenericReport
	ID          string    `xml:"key"`
	Date        string    `xml:"created"`
	Title       string    `xml:"title"`
	Product     string    `xml:"project"`
	Version     string    `xml:"version"`
	Severity    string    `xml:"priority"`
	Reporter    string    `xml:"reporter"`
	Assignee    string    `xml:"assignee"`
	Description string    `xml:"description"`
	Comments    []Comment `xml:"comments>comment"`
}

// New parses an XML file from a bugzilla system
func New(filePath string, url string) (*JiraReport, error) {
	r := JiraReport{}
	r.GenericReport = pogo.GenericReport{FilePath: filePath, Url: url}
	r.DownloadFile()
	r.ParseXML("item", &r)
	r.GenericReport.AllText = func() string {
		return r.AllText(24)
	}
	r.GenericReport.ID = r.ID
	r.GenericReport.Product = r.Product
	return &r, nil
}

func (report JiraReport) AllText(hours float64) string {

	str := report.Title + " " + report.Description

	dateReport, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", report.Date)

	for index := 0; index < len(report.Comments); index++ {

		dateComment, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", report.Comments[index].Date)

		if dateComment.Sub(dateReport).Hours() < hours {
			str += " " + report.Comments[index].Text
		}
	}

	return str
}

func (report JiraReport) String() string {
	var str = "{ID=" + report.ID + "}\n" +
		"{Date=" + report.Date + "}\n" +
		"{Title=" + report.Title + "}\n" +
		"{Product=" + report.Product + "}\n" +
		"{Version=" + report.Version + "}\n" +
		"{Severity=" + report.Severity + "}\n" +
		"{Reporter=" + report.Reporter + "}\n" +
		"{Assignee=" + report.Assignee + "}\n" +
		"{Description=}" + report.Description + "}"

	for index := 0; index < len(report.Comments); index++ {

		str += "\n{COMMENT={\n" +
			"\t {Commenter=" + report.Comments[index].Commenter + "}\n" +
			"\t {Date=" + report.Comments[index].Date + "}\n" +
			"\t {Text=" + report.Comments[index].Text + "}\n" +
			"}\n"
	}

	return str
}
