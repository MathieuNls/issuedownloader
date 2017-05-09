package jira

import (
	"database/sql"
	"log"
	"time"

	"github.com/mathieunls/deepchange-downloader/pogo"
)

// Report represents a bugzilla bug
type Report struct {
	pogo.GenericReport
	ID          string    `xml:"key"`
	Date        string    `xml:"created"`
	DateClosed  string    `xml:"closed"`
	Type        string    `xml:"type"`
	Title       string    `xml:"title"`
	Product     string    `xml:"project"`
	Version     string    `xml:"version"`
	Severity    string    `xml:"priority"`
	Reporter    string    `xml:"reporter"`
	Assignee    string    `xml:"assignee"`
	Description string    `xml:"description"`
	Comments    []Comment `xml:"comments>comment"`
}

// NewSQL fetches information from a SQL database
func NewSQL(db *sql.DB, projectKey string, id string, databaseName string) (*Report, error) {
	r := Report{}
	r.ID = ""
	r.GenericReport = pogo.GenericReport{}

	rows, err := db.Query(`SELECT 
		jiraissue.ID AS ID,
		jiraissue.REPORTER,
		jiraissue.ASSIGNEE,
		jiraissue.SUMMARY,
		jiraissue.DESCRIPTION,
		jiraissue.PRIORITY,
		jiraissue.CREATED,
		jiraissue.UPDATED,
		jiraissue.RESOLUTIONDATE,
		jiraissue.issuenum AS EXTERNAL_ID,
		issuetype.pname AS ISSUE_TYPE,
		jiraaction.AUTHOR AS COMMENT_AUTHOR,
		jiraaction.CREATED AS COMMENT_DATE,
		jiraaction.actionbody AS COMMENT
	FROM
		jiraissue
			JOIN
		issuetype ON issuetype.ID = jiraissue.issuetype
			JOIN
		project ON project.ID = jiraissue.PROJECT 
			AND project.pkey = '`+projectKey+`' 
			LEFT JOIN
		jiraaction ON jiraissue.ID = jiraaction.issueid
			AND jiraaction.actiontype = 'comment'
	WHERE
		jiraissue.issuenum = ?`, id)

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {

		var (
			ID             string
			REPORTER       string
			ASSIGNEE       string
			SUMMARY        string
			DESCRIPTION    string
			PRIORITY       string
			CREATED        string
			UPDATED        string
			RESOLUTIONDATE string
			EXTERNALID     string
			ISSUETYPE      string
			COMMENTAUTHOR  string
			COMMENTDATE    string
			COMMENT        string
		)

		err := rows.Scan(
			&ID,
			&REPORTER,
			&ASSIGNEE,
			&SUMMARY,
			&DESCRIPTION,
			&PRIORITY,
			&CREATED,
			&UPDATED,
			&RESOLUTIONDATE,
			&EXTERNALID,
			&ISSUETYPE,
			&COMMENTAUTHOR,
			&COMMENTDATE,
			&COMMENT)

		if err != nil {
			log.Fatal(err)
		}

		if r.ID == "" {
			r.ID = ID
			r.Reporter = REPORTER
			r.Assignee = ASSIGNEE
			r.Title = SUMMARY
			r.Description = DESCRIPTION
			r.Severity = PRIORITY
			r.Date = CREATED
			r.DateClosed = RESOLUTIONDATE
			r.Type = ISSUETYPE
		}

		r.Comments = append(r.Comments, Comment{
			Commenter: COMMENTAUTHOR,
			Date:      COMMENTDATE,
			Text:      COMMENT})

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return &r, nil
}

// NewXML parses an XML file from a bugzilla system
func NewXML(filePath string, url string) (*Report, error) {
	r := Report{}
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

func (report Report) AllText(hours float64) string {

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

func (report Report) String() string {
	var str = "{ID=" + report.ID + "}\n" +
		"{Date=" + report.Date + "}\n" +
		"{Title=" + report.Title + "}\n" +
		"{Product=" + report.Product + "}\n" +
		"{Version=" + report.Version + "}\n" +
		"{Severity=" + report.Severity + "}\n" +
		"{Reporter=" + report.Reporter + "}\n" +
		"{Assignee=" + report.Assignee + "}\n" +
		"{Description=" + report.Description + "}"

	for index := 0; index < len(report.Comments); index++ {

		str += "\n{COMMENT={\n" +
			"\t {Commenter=" + report.Comments[index].Commenter + "}\n" +
			"\t {Date=" + report.Comments[index].Date + "}\n" +
			"\t {Text=" + report.Comments[index].Text + "}\n" +
			"}\n"
	}

	return str
}
