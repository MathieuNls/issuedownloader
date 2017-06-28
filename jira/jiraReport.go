package jira

import (
	"database/sql"
	"log"
	"time"

	"strings"

	"github.com/mathieunls/deepchange-downloader/pogo"
)

// Report represents a jira bug
type Report struct {
	pogo.ReportAttributes
	Comments []*Comment
}

//MySQLJiraLinker links Jira report based on MYSQL cnx
type MySQLJiraLinker struct {
	Db           *sql.DB
	ProjectKey   string
	DatabaseName string
}

//XMLJiraLinker links Jira report based on XML API
type XMLJiraLinker struct {
	filePath string
	url      string
}

//Fetch fetches a report using mysql
//It expects ids to look like ACE-234430
func (linker *MySQLJiraLinker) Fetch(id string) (pogo.Report, error) {

	id = strings.Split(id, "-")[1]

	report, err := NewSQL(linker.Db, linker.ProjectKey, id, linker.DatabaseName)

	return report, err
}

func (linker *MySQLJiraLinker) DBName() string {
	return linker.DatabaseName
}

func (report *Report) Attributes() *pogo.ReportAttributes {
	return &report.ReportAttributes
}

// func (linker *XMLJiraLinker) Fetch(id string) (*Report, error) {

// 	return NewXML(linker.filePath, linker.url+id)
// }

// NewSQL fetches information from a SQL database
func NewSQL(db *sql.DB, projectKey string, id string, databaseName string) (*Report, error) {
	r := Report{}

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
			AND project.ORIGINALKEY = '`+projectKey+`' 
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
			DESCRIPTION    sql.NullString
			PRIORITY       sql.NullString
			CREATED        string
			UPDATED        string
			RESOLUTIONDATE string
			EXTERNALID     string
			ISSUETYPE      string
			COMMENTAUTHOR  sql.NullString
			COMMENTDATE    sql.NullString
			COMMENT        sql.NullString
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
			return nil, err
		}

		if r.ExternalID == "" {
			r.ExternalID = databaseName + "_" + id
			r.Reporter = REPORTER
			r.Assignee = ASSIGNEE
			r.Title = SUMMARY
			r.Description = DESCRIPTION.String
			r.Severity = PRIORITY.String
			r.Date = CREATED
			r.DateClosed = RESOLUTIONDATE
			r.Type = ISSUETYPE
		} else {
			attr := pogo.CommentAttribut{
				Commenter: COMMENTAUTHOR.String,
				Date:      COMMENTDATE.String,
				Text:      COMMENT.String}

			r.ReportAttributes.Comments = append(r.ReportAttributes.Comments, attr)
		}

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return &r, nil
}

// NewXML parses an XML file from a jira system
// func NewXML(filePath string, url string) (*Report, error) {
// 	r := Report{}
// 	r.GenericReport = pogo.GenericReport{FilePath: filePath, Url: url}
// 	r.DownloadFile()
// 	r.ParseXML("item", &r)
// 	r.GenericReport.AllText = func() string {
// 		return r.AllText(24)
// 	}

// 	intID, _ := strconv.Atoi(r.ID)
// 	r.GenericReport.ID = int64(intID)
// 	r.GenericReport.Product = r.Product
// 	return &r, nil
// }

//AllText returns all the text from the report `hours` after openning
func (report *Report) AllText(hours float64) string {

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

//String returns a string representation
func (report *Report) String() string {
	var str = "{Date=" + report.Date + "}\n" +
		"{Title=" + report.Title + "}\n" +
		"{Product=" + report.Product + "}\n" +
		"{Version=" + report.Version + "}\n" +
		"{Severity=" + report.Severity + "}\n" +
		"{Reporter=" + report.Reporter + "}\n" +
		"{Assignee=" + report.Assignee + "}\n" +
		"{Description=" + report.Description + "}"

	for index := 0; index < len(report.Comments); index++ {

		str += report.Comments[index].String()
	}

	return str
}
