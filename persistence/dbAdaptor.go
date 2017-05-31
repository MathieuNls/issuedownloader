package persistence

import "github.com/mathieunls/deepchange-downloader/pogo"

type DBAdaptor interface {
	SyncCommit(*pogo.Commit)
	IsBuggy(*pogo.Commit, int)
	SyncReports(reports []pogo.Report, repoID int, commitHash string)
	IsLinked(*pogo.Commit, int)
}

var sqlCommitInsert = `INSERT INTO bumper.commit
						(
							hash,
							text,
							is_buggy,
							is_linked,
							subsystems,
							directories,
							files,
							entrophy,
							line_added,
							line_deleted,
							line_total,
							devs,
							age,
							unique_change,
							experience,
							relative_experience,
							subsystem_experience,
							P4_path,
							P4_CL,
							author_id,
							repository_id,
							timestamp
						)
						VALUES
						(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

var sqlPeopleSelectByEmail = `SELECT id
							  FROM people
							  where email = ?
							  LIMIT 1`

var sqlPeopleInsert = `INSERT INTO people
						(
							lastname,
							firstname,
							email,
							sso_id
						)
						VALUES
						(?, ?, ?, ?);`

var sqlWordSelect = `Select id FROM word where word = ? and gram = ? LIMIT 1`

var sqlInsertWord = `INSERT INTO WORD (word, gram) VALUES `

var sqlInsertReport = `INSERT INTO report
						(
							open_at,
							closed_at,
							title,
							description,
							repo_id,
							severity_id,
							reporter_id,
							assignee_id,
							external_id)
						VALUES
						(
							?,
							?,
							?,
							?,
							?,
							?,
							?,
							?,
							?);`

var sqlInsertWordIntermediary = `INSERT INTO TMP_TABLE VALUES `

var sqlInsertComment = `INSERT INTO comment
						(
							commenter_id,
							commented_at,
							text,
							report_id
						)
						VALUES
						(
							?,
							?,
							?,
							?
						);`

var sqlSelectFile = `Select id from file where name = ? and repository_id = ? LIMIT 1`

var sqlInsertFile = `INSERT INTO file
					(
					name,
					repository_id)
					VALUES `

var sqlInsertFileCommit = `INSERT INTO commit_file
							(commit_id,
							file_id) VALUES `

var sqlInsertReviewer = `INSERT INTO commit_reviewer
						(commit_id,
						reviewer_id)
						VALUES
						(?,?);`

var sqlInsertClassification = `INSERT INTO commit_classification
								(commit_id,
								classification_id,
								confidence)
								VALUES
								(?,?,?);`

var sqlSeveritySelect = `Select id FROM severity where description = ? LIMIT 1`

var sqlInsertSeverity = `INSERT INTO severity
						(description)
						VALUES
						(?);`

var sqlFindCommit = `Select id FROM commit where hash = ? and repository_id = ? LIMIT 1`

var sqlUpdateBuggyCommit = `UPDATE commit
							SET
							is_buggy = true,
							WHERE id = ?;`

var sqlUpdateLinkedCommit = `UPDATE commit
							SET
							is_linked = true
							WHERE id = ?;`

var sqlInsertFix = `INSERT INTO commit_fix
					(
					buggy_commit_id,
					fixing_commit_id)
					VALUES
					(?, ?);`

var sqlInsertCommitReport = `Insert into commit_report (commit_id, report_id) VALUES`
