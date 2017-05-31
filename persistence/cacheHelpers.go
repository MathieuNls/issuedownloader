package persistence

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/mathieunls/deepchange-downloader/pogo"
)

//Following structs as used as database cache
type peopleStruct struct {
	ID        int64
	Email     string
	Lastname  string
	Firstname string
	SsoID     string
}

type wordStruct struct {
	ID        int64
	Word      string
	Gram      int
	Frequency int
}

type fileStruct struct {
	ID     int64
	File   string
	RepoID int
}

type severityStruct struct {
	ID       int64
	Severity string
}

type commit struct {
	ID     int64
	Hash   string
	RepoID int
}

func findPeople(email string, lastname string, firstname string, ssoID string, Db *sql.DB) int64 {

	if people := GetCacheInstance().Fetch("people", email); people != nil {
		return people.(*peopleStruct).ID
	}

	var peopleID int64
	stmSel, err := Db.Prepare(sqlPeopleSelectByEmail)
	if err != nil {
		panic(err.Error())
	}
	err = stmSel.QueryRow(email).Scan(&peopleID)
	stmSel.Close()
	switch {
	case err == sql.ErrNoRows:

		stmIns, err := Db.Prepare(sqlPeopleInsert)
		result, err := stmIns.Exec(
			lastname,
			firstname,
			email,
			ssoID)
		if err != nil {
			panic(err.Error())
		}

		peopleID, err = result.LastInsertId()
		stmIns.Close()

	case err != nil:
		log.Fatal(err)
	}

	cachePeople(peopleID, email, lastname, firstname, ssoID)

	return peopleID
}

func findWords(words []*wordStruct, Db *sql.DB) []*wordStruct {

	var cachedWords []*wordStruct
	var unknownWords []*wordStruct

	for _, word := range words {
		if cachedWord := GetCacheInstance().FetchWithCB(
			"word",
			strings.Join([]string{word.Word, strconv.Itoa(word.Gram)}, ""),
			func(tmpWord interface{}) interface{} {
				tmpWord.(*wordStruct).Frequency = word.Frequency
				return tmpWord
			}); cachedWord != nil {
			cachedWords = append(cachedWords, cachedWord.(*wordStruct))
		} else {
			unknownWords = append(unknownWords, word)
		}
	}

	//Do we have unknwon words ?
	if len(unknownWords) > 0 {
		//Construct the sql query for the unknwon files
		sqlStr := sqlInsertWord
		for _, word := range unknownWords {
			sqlStr += "('" + word.Word + "'," + strconv.Itoa(word.Gram) + "),"
		}
		//trim the last ,
		sqlStr = sqlStr[0 : len(sqlStr)-1]
		result, err := Db.Exec(sqlStr)

		//LastID returns the first id inserted in the bulk
		lastID, err := result.LastInsertId()
		rows, err := result.RowsAffected()

		if err != nil {
			panic(err.Error())
		}

		//iterates on the created ids and cache the created files
		for id, unknownWordID := lastID, 0; id <= rows; id++ {
			unknownWords[unknownWordID].ID = id
			GetCacheInstance().Put("word",
				strings.Join(
					[]string{unknownWords[unknownWordID].Word,
						strconv.Itoa(unknownWords[unknownWordID].Gram)},
					""),
				unknownWords[unknownWordID])
			cachedWords = append(cachedWords, unknownWords[unknownWordID])
			unknownWordID++
		}
	}

	return cachedWords
}

func findFiles(files []string, repoID int, Db *sql.DB) []int64 {

	var cachedFiles []int64
	var unknownFiles []string

	//Get known files in cache
	//& populate unknownFiles with the remaining ones
	for _, file := range files {
		if cachedFile := GetCacheInstance().Fetch("file", strings.Join([]string{file, strconv.Itoa(repoID)}, "")); cachedFile != nil {
			cachedFiles = append(cachedFiles, cachedFile.(*fileStruct).ID)
		} else {
			unknownFiles = append(unknownFiles, file)
		}
	}

	//Do we have unknwon files ?
	if len(unknownFiles) > 0 {
		//Construct the sql query for the unknwon files
		sqlStr := sqlInsertFile
		for _, file := range unknownFiles {
			sqlStr += "('" + file + "'," + strconv.Itoa(repoID) + "),"
		}
		//trim the last ,
		sqlStr = sqlStr[0 : len(sqlStr)-1]

		result, err := Db.Exec(sqlStr)

		if err != nil {
			panic(err.Error())
		}

		//LastID returns the first id inserted in the bulk
		lastID, err := result.LastInsertId()
		rows, err := result.RowsAffected()

		if err != nil {
			panic(err.Error())
		}

		//iterates on the created ids and cache the created files
		for id, unknownFileID := lastID, 0; id <= rows; id++ {
			cachedFiles = append(cachedFiles, id)
			GetCacheInstance().Put("file", strings.Join([]string{unknownFiles[unknownFileID], strconv.Itoa(repoID)}, ""), &fileStruct{
				ID:     id,
				File:   unknownFiles[unknownFileID],
				RepoID: repoID,
			})
			unknownFileID++
		}
	}

	return cachedFiles
}

func findSeverity(severity string, Db *sql.DB) int64 {

	if cachedSeverity := GetCacheInstance().Fetch("severity", severity); cachedSeverity != nil {
		return cachedSeverity.(*severityStruct).ID
	}

	var severityID int64
	stmSel, err := Db.Prepare(sqlSeveritySelect)
	if err != nil {
		panic(err.Error())
	}
	err = stmSel.QueryRow(severity).Scan(&severityID)
	stmSel.Close()
	switch {
	case err == sql.ErrNoRows:

		stmIns, err := Db.Prepare(sqlInsertSeverity)
		result, err := stmIns.Exec(severity)
		if err != nil {
			panic(err.Error())
		}

		severityID, err = result.LastInsertId()
		stmIns.Close()

	case err != nil:
		log.Fatal(err)
	}

	cacheSeverity(severity, severityID)

	return severityID
}

func findCommit(hash string, repoID int, Db *sql.DB) int64 {

	if cachedCommit := GetCacheInstance().Fetch("commit", strings.Join([]string{hash, strconv.Itoa(repoID)}, "")); cachedCommit != nil {
		return cachedCommit.(int64)
	}

	var commitID int64
	stmSel, err := Db.Prepare(sqlFindCommit)
	if err != nil {
		panic(err.Error())
	}
	err = stmSel.QueryRow(hash, repoID).Scan(&commitID)
	stmSel.Close()
	if err != nil {
		log.Fatal(err)
	}

	cacheCommit(commitID, hash, repoID)

	return commitID
}

// func findReport(string externalId, Db *sql.DB) int64 {

// }

func cacheFile(file string, repoID int, fileID int64) {
	GetCacheInstance().Put("file", strings.Join([]string{file, strconv.Itoa(repoID)}, ""), &fileStruct{
		ID:     fileID,
		File:   file,
		RepoID: repoID,
	})
}

func cachePeople(
	peopleID int64, email string,
	lastname string, firstname string,
	ssoID string) {

	GetCacheInstance().Put("people", email, &peopleStruct{
		ID:        peopleID,
		Email:     email,
		Lastname:  lastname,
		Firstname: firstname,
		SsoID:     ssoID,
	})
}

func cacheWord(word string, gram int, wordID int64) {
	GetCacheInstance().Put("word", strings.Join([]string{word, strconv.Itoa(gram)}, ""), &wordStruct{
		ID:   wordID,
		Word: word,
		Gram: gram,
	})
}

func cacheSeverity(severity string, severityID int64) {
	GetCacheInstance().Put("severity", severity, &severityStruct{
		ID:       severityID,
		Severity: severity,
	})
}

func cacheCommit(commitID int64, hash string, repoID int) {
	GetCacheInstance().Put("commit", strings.Join([]string{hash, strconv.Itoa(repoID)}, ""), commitID)
}

func cacheReport(externalID string, reportID int64) {

	attr := pogo.ReportAttributes{
		ID:         reportID,
		ExternalID: externalID,
	}

	GetCacheInstance().Put("report", externalID, attr)
}

func WarmupCache(Db *sql.DB, logDirs []string) {

	fmt.Println("Warmin up cache")

	queries := []string{
		"Select * from file",
		"Select * from people",
		"Select id, hash, repository_id from commit",
		"Select * from word",
		"Select * from severity",
		"Select external_id, id from report",
	}

	cachingFunctions := []func(rows *sql.Rows){
		func(rows *sql.Rows) {
			var file string
			var repoID int
			var fileID int64

			rows.Scan(&fileID, &file, &repoID)
			cacheFile(file, repoID, fileID)
		},
		func(rows *sql.Rows) {
			var id int64
			var lastname, firstname, email, ssoID string

			rows.Scan(&id, &lastname, &firstname, &email, &ssoID)
			cachePeople(id, email, lastname, firstname, ssoID)
		},
		func(rows *sql.Rows) {
			var id int64
			var repoID int
			var hash string

			rows.Scan(&id, &hash, &repoID)
			cacheCommit(id, hash, repoID)
		},
		func(rows *sql.Rows) {
			var id int64
			var word string
			var gram int

			rows.Scan(&id, &word, &gram)
			cacheWord(word, gram, id)
		},
		func(rows *sql.Rows) {
			var id int64
			var description string

			rows.Scan(&id, &description)
			cacheSeverity(description, id)
		},
		func(rows *sql.Rows) {
			var id int64
			var externalID string

			rows.Scan(&externalID, &id)
			cacheReport(externalID, id)
		},
	}

	for i := 0; i < len(queries); i++ {

		fmt.Println("...", queries[i])

		rows, err := Db.Query(queries[i])

		if err != nil {
			panic(err.Error())
		}

		for rows.Next() {

			cachingFunctions[i](rows)
		}

		rows.Close()
	}

	for _, logDir := range logDirs {
		fmt.Println("... files in", logDir)

		files, err := ioutil.ReadDir(logDir)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {

			store := ""

			if strings.Index(file.Name(), "diff_") == 0 {
				store = "diff"
			} else if strings.Index(file.Name(), "file_modified_diff_") == 0 {
				store = "file_modified_diff"
			} else if strings.Index(file.Name(), "blame_") == 0 {
				store = "blame"
			}

			content, err := ioutil.ReadFile(logDir + file.Name())

			if store != "" && err == nil {
				GetCacheInstance().Put(store, file.Name(), content)
			} else {

				log.Panic(file.Name(), "\n", store, "\n", file.Name(), "\n", content)
				panic(err)

			}
		}
	}

}
