package bugzilla

import (
	"strconv"
	"time"

	"github.com/BUMPER/IssueDownloader/pogo"
)

// BzReport represents a bugzilla bug
type BzReport struct {
	pogo.GenericReport
	ID        string    `xml:"bug_id"`
	Date      string    `xml:"creation_ts"`
	Title     string    `xml:"short_desc"`
	Product   string    `xml:"product"`
	Component string    `xml:"component"`
	Version   string    `xml:"version"`
	Platform  string    `xml:"rep_platform"`
	OS        string    `xml:"op_sys"`
	Severity  string    `xml:"bug_severity"`
	Reporter  string    `xml:"reporter"`
	Assignee  string    `xml:"assigned_to"`
	Comments  []Comment `xml:"long_desc"`
}

// New parses an XML file from a bugzilla system
func New(filePath string, url string) (*BzReport, error) {
	r := BzReport{}
	r.GenericReport = pogo.GenericReport{FilePath: filePath, Url: url}
	r.DownloadFile()
	r.ParseXML("bug", &r)
	r.GenericReport.AllText = func() string {
		return r.AllText(24.0)
	}
	r.GenericReport.ID = r.ID
	r.GenericReport.Product = r.Product
	return &r, nil
}

func (report BzReport) AllText(hours float64) string {

	str := report.Title

	dateReport, _ := time.Parse("2006-01-02 15:04:05 -0700", report.Date)

	for index := 0; index < len(report.Comments); index++ {
		dateComment, _ := time.Parse("2006-01-02 15:04:05 -0700", report.Comments[index].Date)

		if dateComment.Sub(dateReport).Hours() < hours {
			str += " " + report.Comments[index].Text
		}

	}

	return str
}

func (report BzReport) String() string {
	var str = "{ID=" + report.ID + "}\n" +
		"{Date=" + report.Date + "}\n" +
		"{Title=" + report.Title + "}\n" +
		"{Product=" + report.Product + "}\n" +
		"{Component=" + report.Component + "}\n" +
		"{Version=" + report.Version + "}\n" +
		"{Platform=" + report.Platform + "}\n" +
		"{OS=" + report.OS + "}\n" +
		"{Severity=" + report.Severity + "}\n" +
		"{Reporter=" + report.Reporter + "}\n" +
		"{Assignee=" + report.Assignee + "}\n"

	for index := 0; index < len(report.Comments); index++ {

		str += "\n{COMMENT={\n" +
			"\t {Commenter=" + report.Comments[index].Commenter + "}\n" +
			"\t {Order=" + strconv.Itoa(report.Comments[index].Order) + "}\n" +
			"\t {Date=" + report.Comments[index].Date + "}\n" +
			"\t {Text=" + report.Comments[index].Text + "}\n" +
			"}\n"
	}

	return str
}
