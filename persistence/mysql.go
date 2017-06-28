package persistence

import (
	"database/sql"
	"fmt"
	"strconv"

	"strings"

	"github.com/mathieunls/deepchange-downloader/helper"
	"github.com/mathieunls/deepchange-downloader/pogo"
	"github.com/mathieunls/deepchange-downloader/wordnet"
	gcache "github.com/mathieunls/gcache/src"
)

//MySQLAdaptor retuns a MySQLAdaptor
type MySQLAdaptor struct {
	Db           *sql.DB
	DatabaseName string
	Gram         int
	Cache        gcache.Cache
	nbCommit     int
}

//SyncCommit sync commit
func (mysql *MySQLAdaptor) SyncCommit(commit *pogo.Commit) {
	mysql.nbCommit++
	fmt.Println("Saving commit", commit.CommitHash, "("+strconv.Itoa(mysql.nbCommit)+")")

	stmIns, err := mysql.Db.Prepare(sqlCommitInsert)
	if err != nil {
		panic(err.Error())
	}

	result, err := stmIns.Exec(
		commit.CommitHash,
		helper.UTF8String(commit.CommitMessage),
		commit.ContainsBug,
		commit.Linked,
		commit.Subsystems,
		commit.Directories,
		commit.Files,
		commit.Entrophy,
		commit.LineAdded,
		commit.LineDeleted,
		commit.LineTotal,
		commit.Devs,
		commit.Age,
		commit.UniqueChange,
		commit.Exp,
		commit.RExp,
		commit.Sexp,
		commit.P4Path,
		commit.P4CL,
		findPeople(helper.UTF8String(commit.AuthorEmail), helper.UTF8String(commit.AuthorName), "", "", mysql.Db),
		commit.RepositoryID,
		commit.AuthorDateUnixTimestamp)

	if err != nil {

		fmt.Println(commit)
		panic(err.Error())
	}

	commit.ID, err = result.LastInsertId()

	mysql.syncFiles(commit.FilesChanged, commit.ID, commit.RepositoryID)

	mysql.insertWords(func(gram int) map[string]int {
		fmt.Println(".. Saving commit", commit.CommitHash, "'s words", gram, "grams")
		return wordnet.ExtractUniqGrams(commit.CommitMessage, gram)
	}, "commit_word", commit.ID)

	mysql.insertClassifications(commit.Classification, commit.ID)

	fmt.Println(".. Saving commit", commit.CommitHash, "'s reviewers", len(commit.Reviewers))

	for _, reviewer := range commit.Reviewers {
		reviewerID := findPeople(reviewer, reviewer, "", "", mysql.Db)
		stmInsReviewer, errReviewer := mysql.Db.Prepare(sqlInsertReviewer)
		if errReviewer != nil {
			panic(errReviewer.Error())
		}

		_, errReviewer = stmInsReviewer.Exec(commit.ID, reviewerID)

		if errReviewer != nil {
			panic(errReviewer.Error())
		}

		stmInsReviewer.Close()
	}

	if err != nil {
		panic(err.Error())
	}

	stmIns.Close()
}

func (mysql *MySQLAdaptor) insertClassifications(classifications map[string]float64, commitID int64) {

	fmt.Println(".. Saving commit's classification")

	for classification, percentage := range classifications {

		if percentage > 0.0 {
			stmIns, err := mysql.Db.Prepare(sqlInsertClassification)
			if err != nil {
				panic(err.Error())
			}

			classificationID := -1

			switch classification {
			case "corrective":
				classificationID = 1
			case "feature_addition":
				classificationID = 2
			case "non_functional":
				classificationID = 3
			case "perfective":
				classificationID = 4
			case "preventive":
				classificationID = 5
			case "merge":
				classificationID = 6
			}

			_, err = stmIns.Exec(commitID, classificationID, percentage)

			if err != nil {
				panic(err.Error())
			}

			stmIns.Close()
		}

	}

}

func (mysql *MySQLAdaptor) syncFiles(filesChanged []string, commitID int64, repositoryID int) {

	fmt.Println(".. Saving commit's files", len(filesChanged))

	if len(filesChanged) > 0 {
		fileIDs := findFiles(filesChanged, repositoryID, mysql.Db)

		if len(fileIDs) > 0 {
			sqlStr := sqlInsertFileCommit
			for _, fileID := range fileIDs {

				sqlStr += "(" + strconv.Itoa(int(commitID)) + "," + strconv.Itoa(int(fileID)) + "),"
			}

			//trim the last ,
			sqlStr = sqlStr[0 : len(sqlStr)-1]

			_, errFile := mysql.Db.Exec(sqlStr)

			if errFile != nil {
				panic(errFile.Error())
			}
		}
	}
}

func (mysql *MySQLAdaptor) insertWords(words func(int) map[string]int, table string, parentID int64) {

	strSQL := strings.Replace(sqlInsertWordIntermediary, "TMP_TABLE", table, 1)

	var allWords []*wordStruct

	//get all the grams words and their frequency
	for index := 1; index < mysql.Gram+1; index++ {

		for word, frequency := range words(index) {
			allWords = append(allWords, &wordStruct{
				Word:      word,
				Frequency: frequency,
			})
		}
	}

	if len(allWords) > 1 {
		//get ids for words from cache / database
		allWords := findWords(allWords, mysql.Db)

		for _, word := range allWords {
			strSQL += "(" + strconv.Itoa(int(parentID)) +
				", " + strconv.Itoa(int(word.ID)) + ", " +
				strconv.Itoa(word.Frequency) +
				", null),"
		}

		//trim the last ,
		strSQL = strSQL[0 : len(strSQL)-1]

		_, err := mysql.Db.Exec(strSQL)

		if err != nil {
			fmt.Println(strSQL)
			panic(err.Error())
		}
	}
}

//SyncReports sync reports
func (mysql *MySQLAdaptor) SyncReports(reports []pogo.Report, repoID int, commitHash string) {

	commit := findCommit(commitHash, repoID, mysql.Db)

	if len(reports) > 0 {
		sqlStr := sqlInsertCommitReport

		for _, report := range reports {

			//Is that report already locally synced ?
			if report.Attributes().ID == 0 {
				fmt.Println(".. Saving commit's reports", commitHash)
				stmIns, err := mysql.Db.Prepare(sqlInsertReport)
				if err != nil {
					panic(err.Error())
				}

				reportAttributes := report.Attributes()

				result, err := stmIns.Exec(

					reportAttributes.Date,
					reportAttributes.DateClosed,
					helper.UTF8String(reportAttributes.Title),
					helper.UTF8String(reportAttributes.Description),
					repoID,
					findSeverity(reportAttributes.Severity, mysql.Db),
					findPeople(reportAttributes.Reporter, reportAttributes.Reporter, "", "", mysql.Db),
					findPeople(reportAttributes.Assignee, reportAttributes.Assignee, "", "", mysql.Db),
					reportAttributes.ExternalID,
				)

				if err == nil {

					reportAttributes.ID, err = result.LastInsertId()

					if err != nil {
						panic(err.Error())
					}

					stmIns.Close()

					mysql.Cache.Put("report", reportAttributes.ExternalID, *reportAttributes)

					mysql.insertWords(func(gram int) map[string]int {
						return wordnet.ExtractUniqGrams(report.AllText(9999999), gram)
					}, "report_word", reportAttributes.ID)

					mysql.SyncReportsComment(reportAttributes.Comments, reportAttributes.ID)

				} else {

					fmt.Println("SKIPPED")
					fmt.Println(reportAttributes)
					fmt.Println("SKIPPED")
				}

			}

			if report.Attributes().ID != 0 &&
				mysql.Cache.Fetch("report_commit", strconv.Itoa(int(commit.ID))+"-"+strconv.Itoa(int(report.Attributes().ID))) != nil {
				mysql.Cache.Put("report_commit", strconv.Itoa(int(commit.ID))+"-"+strconv.Itoa(int(report.Attributes().ID)), "there")
				sqlStr += "(" + strconv.Itoa(int(commit.ID)) + "," + strconv.Itoa(int(report.Attributes().ID)) + "),"
			}

		}

		if strings.Compare(sqlStr, sqlInsertCommitReport) != 0 {

			//trim the last ,
			sqlStr = sqlStr[0 : len(sqlStr)-1]

			_, errFile := mysql.Db.Exec(sqlStr)

			if errFile != nil {
				fmt.Println(errFile.Error() + "hash: " + commitHash)
				panic(errFile)
			}
		}

	}
}

//SyncReportsComment syncs the reports
func (mysql *MySQLAdaptor) SyncReportsComment(comments []pogo.CommentAttribut, reportID int64) {

	fmt.Println(" Saving comment for report", reportID, len(comments))

	for _, comment := range comments {

		stmIns, err := mysql.Db.Prepare(sqlInsertComment)
		if err != nil {
			panic(err.Error())
		}
		result, err := stmIns.Exec(
			findPeople(comment.Commenter, comment.Commenter, "", "", mysql.Db),
			comment.Date,
			helper.UTF8String(comment.Text),
			reportID)

		if err != nil {
			fmt.Println(comment)
			panic(err)
		}

		commentID, err := result.LastInsertId()

		mysql.insertWords(func(gram int) map[string]int {
			return wordnet.ExtractUniqGrams(comment.Text, gram)
		}, "comment_word", commentID)

		if err != nil {
			panic(err)
		}

		stmIns.Close()
	}
}

//IsBuggy update a change
func (mysql *MySQLAdaptor) IsBuggy(commit *pogo.Commit, repoID int) {

	stmt, err := mysql.Db.Prepare(sqlUpdateBuggyCommit)
	if err != nil {
		panic(err.Error())
	}

	_, err = stmt.Exec(findCommit(commit.CommitHash, repoID, mysql.Db).ID)

	if err != nil {
		panic(err.Error())
	}

	stmt.Close()

	for _, fixingHash := range commit.FixHashes {

		stmt, err = mysql.Db.Prepare(sqlInsertFix)

		if err != nil {
			panic(err.Error())
		}
		_, err = stmt.Exec(
			findCommit(commit.CommitHash, repoID, mysql.Db).ID,
			findCommit(fixingHash, repoID, mysql.Db).ID,
		)
		if err != nil {
			panic(err.Error())
		}

		stmt.Close()
	}

}

func (mysql *MySQLAdaptor) IsLinked(commit *pogo.Commit, repoID int) {

	c := findCommit(commit.CommitHash, repoID, mysql.Db)

	if c.Linked == false {
		stmt, err := mysql.Db.Prepare(sqlUpdateLinkedCommit)
		if err != nil {
			panic(err.Error())
		}

		_, err = stmt.Exec(c.ID)

		if err != nil {
			panic(err.Error())
		}

		stmt.Close()
	}

}
