package git

import (
	"regexp"
	"strconv"
	"strings"

	classifier "github.com/mathieunls/deepchange-downloader/classifiers"
	"github.com/mathieunls/deepchange-downloader/wordnet"
)

// Commit represents a Git commit
type Commit struct {
	RepositoryID            string
	CommitHash              string
	ParentHashes            []string
	AuthorName              string
	AuthorDateUnixTimestamp int
	AuthorEmail             string
	AuthorDate              string
	Reviewers               []string
	CommitMessage           string
	CommitMessageWords      map[string]float32
	Classification          []string
	Linked                  bool
	ContainsBug             bool
	Fixes                   []string
	Subsystems              int
	Directories             int
	Files                   int
	Entrophy                float64
	LineAdded               int
	LineDeleted             int
	FilesChanged            []string
	LineTotal               float64
	Devs                    int
	Age                     float64
	UniqueChange            int
	Exp                     float64
	RExp                    float64
	Sexp                    float64
	GlmProb                 float64
	P4Path                  string
	P4CL                    string
}

//NewCommit proerply handle the construction of a Git Commit
func NewCommit(parentHashes []string, commitHash string, authorName string,
	authorEmail string, authorDate string, authorDateUnixTimestamp string,
	commitMessage string, regexFix string, regexReviewer string, isP4 bool) *Commit {

	commit := Commit{}

	commit.ParentHashes = parentHashes
	commit.CommitHash = commitHash
	commit.AuthorName = authorName
	commit.AuthorEmail = authorEmail
	commit.AuthorDate = authorDate
	commit.CommitMessage = commitMessage
	commit.CommitMessageWords = wordnet.ExtractUniqWords(commit.CommitMessage)

	//trying to parse the unix timestamp
	var err error
	commit.AuthorDateUnixTimestamp, err = strconv.Atoi(authorDateUnixTimestamp)
	if err != nil {
		panic(err)
	}

	if len(commit.ParentHashes) == 2 {
		commit.Classification = []string{"Merge"}
	}

	//Extracts the fixes w/ regards to regexFix
	re := regexp.MustCompile(regexFix)
	resultSlice := re.FindAllStringSubmatch(commit.CommitMessage, -1)
	commit.Fixes = []string{}
	for index := 0; index < len(resultSlice); index++ {
		if len(resultSlice[index]) == 4 {
			commit.Fixes = append(commit.Fixes, resultSlice[index][3])
		} else if len(resultSlice[index]) == 2 {
			commit.Fixes = append(commit.Fixes, resultSlice[index][1])
		}
	}

	//Extracts the reviewers w/ regards to regexReviewer
	commit.Reviewers = []string{}
	re = regexp.MustCompile(regexReviewer)
	resultSlice = re.FindAllStringSubmatch(commit.CommitMessage, -1)
	for index := 0; index < len(resultSlice); index++ {
		if len(resultSlice[index]) == 2 {
			commit.Reviewers = append(commit.Reviewers, strings.Split(resultSlice[index][1], ",")...)
		}
	}

	commit.Classification = classifier.GetInstance().Categorize(commit.CommitMessage)

	//Extract  info if required
	if isP4 {
		re = regexp.MustCompile(`\[git-p4: depot-paths = "([a-zA-Z0-9/-_]+)": change = ([0-9]+)\]`)
		resultSlice = re.FindAllStringSubmatch(commit.CommitMessage, -1)
		for index := 0; index < len(resultSlice); index++ {

			commit.P4Path = resultSlice[index][1]
			commit.P4CL = resultSlice[index][2]
		}
	}

	return &commit
}

//String returns a string representation of a commit
func (c Commit) String() string {

	return "RepositoryID: " + c.RepositoryID + "\n" +
		"CommitHash: " + c.CommitHash + "\n" +
		"ParentHashes: " + strings.Join(c.ParentHashes, ",") + "\n" +
		"AuthorName: " + c.AuthorName + "\n" +
		"AuthorDateUnixTimestamp: " + strconv.Itoa(c.AuthorDateUnixTimestamp) + "\n" +
		"AuthorEmail: " + c.AuthorEmail + "\n" +
		"AuthorDate: " + c.AuthorDate + "\n" +
		"Reviewers: " + strings.Join(c.Reviewers, ",") + "\n" +
		"CommitMessage: " + c.CommitMessage + "\n" +
		"Classification: " + strings.Join(c.Classification, ",") + "\n" +
		"Linked: " + strconv.FormatBool(c.Linked) + "\n" +
		"ContainsBug: " + strconv.FormatBool(c.ContainsBug) + "\n" +
		"Fixes: " + strings.Join(c.Fixes, ",") + "\n" +
		"Subsystems: " + strconv.Itoa(c.Subsystems) + "\n" +
		"Directories: " + strconv.Itoa(c.Directories) + "\n" +
		"Files: " + strconv.Itoa(c.Files) + "\n" +
		"Entrophy: " + strconv.FormatFloat(c.Entrophy, 'f', 6, 64) + "\n" +
		"LineAdded: " + strconv.Itoa(c.LineAdded) + "\n" +
		"LineDeleted: " + strconv.Itoa(c.LineDeleted) + "\n" +
		"FilesChanged: " + strings.Join(c.FilesChanged, ",") + "\n" +
		"LineTotal: " + strconv.FormatFloat(c.LineTotal, 'f', 6, 64) + "\n" +
		"Devs: " + strconv.Itoa(c.Devs) + "\n" +
		"Age: " + strconv.FormatFloat(c.Age, 'f', 6, 64) + "\n" +
		"UniqueChange: " + strconv.Itoa(c.UniqueChange) + "\n" +
		"Exp: " + strconv.FormatFloat(c.Exp, 'f', 6, 64) + "\n" +
		"RExp: " + strconv.FormatFloat(c.RExp, 'f', 6, 64) + "\n" +
		"Sexp: " + strconv.FormatFloat(c.Sexp, 'f', 6, 64) + "\n" +
		"GlmProb: " + strconv.FormatFloat(c.GlmProb, 'f', 6, 64) + "\n" +
		"P4CL: " + c.P4CL + "\n" +
		"P4Path: " + c.P4Path + "\n"
}
