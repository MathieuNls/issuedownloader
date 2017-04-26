package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	_ "github.com/mathieunls/deepchange-downloader/pogo"
)

// CMD is an abstraction of diverse git commands
type CMD struct {
	logformat         string
	cloneCMD          string
	pullCMD           string
	resetCMD          string
	cleanCMD          string
	headCommitHashCMD string
}

// commitFile is an internal representation of
// a commit file for git commands
type commitFile struct {
	Name        string
	LOC         int
	Authors     map[string]struct{}
	Lastchanged int
	NUC         int
}

// devExperiences is an internal representation
// of developers experiences on project's subsystem
type devExperiences struct {
	systems map[string]int
}

// New returns a new GitCMD abstraction
func New() *CMD {

	g := CMD{}

	//A commit mesasge in git is done such that first line is treated as the subject,
	//and the rest is treated as the message. We combine them under field commit_message
	//We want the log in ascending order, so we call --reverse
	//Numstat is used to get statistics for each commit
	g.logformat = `--pretty=format:BUMPER_STARTPRETTY%P BUMPER_DELIMITER2 %H BUMPER_DELIMITER2 %an BUMPER_DELIMITER2 %ae BUMPER_DELIMITER2 %ad BUMPER_DELIMITER2 %at BUMPER_DELIMITER2 %s%bBUMPER_STOPPRETTY`

	// git clone command w/o downloading src code
	g.cloneCMD = "git clone {!s} {!s}"
	// git pull command
	g.pullCMD = "git pull"
	g.resetCMD = "git reset --hard HEAD"
	//# f for force clean, d for untracked directories
	g.cleanCMD = "git clean -df"
	g.headCommitHashCMD = "git rev-parse HEAD"

	return &g
}

// commitStats extracts the statistics for a commit with
// regards to previous commits
func (git *CMD) commitStats(
	//A string array comming from --numstats
	//It contains strings like "1       2       cluster.R"
	stats []string,
	//All the commitFile we've seen before
	//Map are pointer type. Modifications in here
	//affects the caller
	commitFiles map[string]commitFile,
	//All the experiences of all the devs
	//Once again, a pointer.
	devsExp map[string]devExperiences,
	//Unique name of commiter
	author string,
	//The timestamp (i.e. 1406214540)
	unixTimeStamp int,
	//A pointer to the commit to update
	commit *Commit) {

	//Following maps keep references of known authors,
	//subsystems, directories and files; respectively.
	//The map[string]struct{} is used to build a dictionary
	//without needing additional space: struct{} cost nothing
	authors := make(map[string]struct{})
	subsystems := make(map[string]struct{})
	directories := make(map[string]struct{})
	files := make(map[string]struct{})

	//Commit wide stats
	la := 0     // lines added
	ld := 0     // lines deleted
	nf := 0.0   //number of files
	age := 0.0  //age of file
	exp := 0.0  //experience of dev
	rexp := 0.0 //relative experience with regards to file age
	sexp := 0.0 //subsystem experience
	nuc := 0    //number of unique modification
	lt := 0     //total line in the file before the commit

	totalLOCModified := 0

	//Iterates over all files in the commit, for example
	// 2       0       .gitignore
	// 1       0       actors.json
	// 48      0       cluster.R
	for i := 0; i < len(stats); i++ {

		//Split on spaces
		//2       0       .gitignore
		//becomes [2,0,.gitignore]
		fileStat := strings.Fields(stats[i])

		//discard blank lines
		if len(fileStat) > 0 {

			addeLines, err := strconv.Atoi(fileStat[0])

			if err != nil {
				addeLines = 0
			}

			removedLines, err := strconv.Atoi(fileStat[1])

			if err != nil {
				removedLines = 0
			}

			fileName := fileStat[2]

			totalLOCModified = totalLOCModified + addeLines + removedLines

			//Do we know that author ?
			if _, present := authors[author]; !present {
				authors[author] = struct{}{}
			}

			//We've seen that file already, update stats
			if cFile, present := commitFiles[fileName]; present {

				nuc += cFile.NUC
				lt += cFile.LOC
				age += (float64(unixTimeStamp) - float64(cFile.Lastchanged)) / 86400.0

				cFile.LOC = cFile.LOC + addeLines - removedLines
				cFile.Lastchanged = unixTimeStamp
				cFile.NUC = cFile.NUC + 1
				cFile.Authors[author] = struct{}{}

			} else {

				commitFiles[fileName] = commitFile{fileName, addeLines - removedLines, authors, unixTimeStamp, 1}
			}

			fileDirs := strings.Split(fileName, "/")

			directory := "root"
			subsystem := "root"

			//Are we in a subsystem ?
			if len(fileDirs) > 1 {
				subsystem = fileDirs[0]
				directory = strings.Join(append(fileDirs[:0], fileDirs[1:]...), "/")
			}

			//Do we known that subsystem ?
			if _, present := subsystems[subsystem]; !present {
				subsystems[subsystem] = struct{}{}
			}

			//Do we know that dev XP ?
			if _, present := devsExp[author]; !present {

				devMap := make(map[string]int)
				devMap[subsystem] = 1
				devsExp[author] = devExperiences{devMap}
			} else {

				devExp := devsExp[author]

				for _, val := range devExp.systems {
					exp += float64(val)
				}

				if age != 0 {

					rexp += (1/age + 1)
				}

				if _, present := devsExp[author].systems[subsystem]; !present {
					devsExp[author].systems[subsystem] = 1
				} else {
					sexp += float64(devsExp[author].systems[subsystem])
					devsExp[author].systems[subsystem]++
				}
			}

			//Do we know that dir ?
			if _, present := directories[directory]; !present {
				directories[directory] = struct{}{}
			}

			//Update commit wide stats
			la += addeLines
			ld += removedLines
			nf++
			files[fileName] = struct{}{}

		}

	}

	//Commit had files
	if nf > 0 {

		commit.LineAdded = la
		commit.LineDeleted = ld

		fileSlice := make([]string, len(files))
		i := 0
		for k := range files {
			fileSlice[i] = k
			i++
		}
		commit.FilesChanged = fileSlice
		commit.Files = len(files)
		commit.Subsystems = len(subsystems)
		commit.Directories = len(directories)
		commit.Age = age / nf

		authorSlice := make([]string, len(authors))
		i = 0
		for k := range authors {
			authorSlice[i] = k
			i++
		}
		commit.Devs = len(authorSlice)
		commit.Exp = exp / nf
		commit.RExp = rexp / nf
		commit.Sexp = sexp / nf
		commit.UniqueChange = nuc
		commit.LineTotal = float64(lt) / nf
	}
}

//Commits retuns all the commits of repoDir
func (git *CMD) Commits(repoDir string, lastIngestedCommit string) []*Commit {

	//Run git log
	cmdArgs := []string{"log", "--numstat", "--reverse", git.logformat}

	if lastIngestedCommit != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("%s..HEAD", lastIngestedCommit))
	}

	var cmdOut []byte
	var err error

	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = repoDir

	if cmdOut, err = cmd.Output(); err != nil {
		log.Panic("There was an error running git log command: ", err)
		os.Exit(1)
	}

	commitFiles := make(map[string]commitFile)
	devExp := make(map[string]devExperiences)

	commitList := strings.Split(string(cmdOut), "BUMPER_STARTPRETTY")

	commits := []*Commit{}

	for index := 1; index < len(commitList); index++ {

		prettyCommitSplit := strings.Split(commitList[index], "BUMPER_STOPPRETTY")
		prettyCommit := prettyCommitSplit[0]
		statsCommit := prettyCommitSplit[1]

		prettyCommitDetails := strings.Split(prettyCommit, " BUMPER_DELIMITER2")

		commit := NewCommit(
			strings.Split(prettyCommitDetails[0], " "),
			strings.Trim(prettyCommitDetails[1], " "),
			strings.Trim(prettyCommitDetails[2], " "),
			strings.Trim(prettyCommitDetails[3], " "),
			strings.Trim(prettyCommitDetails[4], " "),
			strings.Trim(prettyCommitDetails[5], " "),
			strings.Trim(prettyCommitDetails[6], " "),
			`@fix(ed\()?( )?([a-zA-Z-]+[0-9]+)`,
			`@review\(([a-z,]+)\)`,
			true)

		stats := strings.Split(statsCommit, "\n")
		git.commitStats(
			stats,
			commitFiles,
			devExp,
			commit.AuthorEmail,
			commit.AuthorDateUnixTimestamp,
			commit)

		fmt.Println(commit)

		commits = append(commits, commit)
	}

	return commits
}

func (git *CMD) linkCorrectiveCommit(commit Commit) []Commit {

	return nil
}

func (git *CMD) getModifiedRegions(commit Commit, repoDir string) []Commit {

	diffCMD := "diff " + commit.CommitHash + "^ " + commit.CommitHash + " --unified=0 " +
		" | while read; do echo \":BUMPER_START:$REPLY:BUMPER_END:\"; done"

	cmd := exec.Command("git", diffCMD)
	cmd.Dir = repoDir

	if cmdOut, err := cmd.Output(); err != nil {
		log.Panic("There was an error running git log command: ", err)
		os.Exit(1)
	}

	return nil

}
