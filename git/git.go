package git

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"io/ioutil"

	classifier "github.com/mathieunls/deepchange-downloader/classifiers"
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
	var cmdOut []byte
	var err error

	pwd, _ := os.Getwd()

	// Log file already exists
	if _, err = os.Stat(pwd + "/data/logs/" + filepath.Base(repoDir) + ".log"); err == nil {

		fmt.Println("Found file", pwd+"/data/logs/"+filepath.Base(repoDir)+".log")
		cmdOut, err = ioutil.ReadFile(pwd + "/data/logs/" + filepath.Base(repoDir) + ".log")

		if err != nil {
			panic(err)
		}
	} else {
		//Run git log
		cmdArgs := []string{"log", "--numstat", "--reverse", git.logformat}

		if lastIngestedCommit != "" {
			cmdArgs = append(cmdArgs, fmt.Sprintf("%s..HEAD", lastIngestedCommit))
		}

		cmd := exec.Command("git", cmdArgs...)
		cmd.Dir = repoDir

		if cmdOut, err = cmd.Output(); err != nil {
			log.Panic("There was an error running git log command: ", err)
			os.Exit(1)
		}

	}

	commitFiles := make(map[string]commitFile)
	devExp := make(map[string]devExperiences)
	commitList := strings.Split(string(cmdOut), "BUMPER_STARTPRETTY")

	commits := []*Commit{}
	trueCorrectiveCommits := []*Commit{}
	correctiveCommits := []*Commit{}
	totalFixReports := 0

	for index := 1; index < len(commitList); index++ {

		fmt.Println(index)

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

		for _, classification := range commit.Classification {
			if strings.Compare(classification, "corrective") == 0 {
				if len(commit.Classification) == 1 {
					trueCorrectiveCommits = append(trueCorrectiveCommits, commit)
				} else {
					correctiveCommits = append(correctiveCommits, commit)
				}

			}
		}

		totalFixReports += len(commit.FixReports)

		commits = append(commits, commit)
	}

	fmt.Println("Commits:", len(commits))
	fmt.Println("Pure Corrective Commits:", len(trueCorrectiveCommits))
	fmt.Println("Corrective Commits:", len(correctiveCommits))
	fmt.Println("Reports closed:", totalFixReports)

	git.linkCorrectiveCommits(correctiveCommits, commits, repoDir)

	return commits
}

func (git *CMD) linkCorrectiveCommits(correctiveCommits []*Commit, allCommits []*Commit, repoDir string) {

	linkedCommits := make(map[string][]string)

	//goroutines  me https://gobyexample.com/mutexes

	for _, correctiveCommit := range correctiveCommits {
		regionChunks := git.getModifiedRegions(*correctiveCommit, repoDir)
		bugIntroducingChanges := git.annotate(regionChunks, *correctiveCommit, repoDir)

		for buggyCommit := range bugIntroducingChanges {

			if _, present := linkedCommits[buggyCommit]; present {

				linkedCommits[buggyCommit] = append(linkedCommits[buggyCommit], correctiveCommit.CommitHash)
			} else {
				linkedCommits[buggyCommit] = []string{correctiveCommit.CommitHash}
			}
		}
		correctiveCommit.Linked = true
	}

	for _, commit := range allCommits {
		if _, present := linkedCommits[commit.CommitHash]; present {
			commit.ContainsBug = true
			commit.FixHashes = linkedCommits[commit.CommitHash]
		}
	}
}

func (git *CMD) getModifiedRegions(commit Commit, repoDir string) map[string][]string {

	cmdArgs := []string{
		"git",
		"diff",
		commit.CommitHash + "^",
		commit.CommitHash,
		"--unified=0",
		" | while",
		"read; do echo \":BUMPER_DELIMITER_START:$REPLY:BUMPER_DELIMITER:\"; done",
	}

	cmd := exec.Command("bash", "-c", strings.Join(cmdArgs, " "))
	cmd.Dir = repoDir

	var diff []byte
	var err error

	if diff, err = cmd.Output(); err != nil {
		log.Panic("There was an error running git diff command: ", err,
			"--- bash ", "-c ", strings.Join(cmdArgs, " "), "at", repoDir)
		os.Exit(1)
	}

	nameOnlyArgs := []string{
		"diff",
		commit.CommitHash + "^",
		commit.CommitHash,
		"--name-only",
	}

	cmd = exec.Command("git", nameOnlyArgs...)
	cmd.Dir = repoDir

	var filesModifiedOUT []byte
	// get the files modified -> use this to validate if we have arrived at a new file
	// when grepping for the specific lines changed.
	if filesModifiedOUT, err = cmd.Output(); err != nil {
		log.Println("There was an error running git diff command: ", err,
			"--- git ", strings.Join(nameOnlyArgs, " "), " at", repoDir, "previous command was",
			"--- bash ", "-c ", strings.Join(cmdArgs, " "), " at", repoDir)
		// os.Exit(1)
	} else {

		filesModified := strings.Split(strings.Replace(string(filesModifiedOUT), "b'", "", -1), "\n")
		return git.extractRegions(string(diff), filesModified)
	}

	return make(map[string][]string)
}

func (git *CMD) extractRegions(diff string, filesModified []string) map[string][]string {

	var regionDiff = make(map[string][]string)

	for _, file := range filesModified {

		// weed out bad files/binary files/etc
		if file != "'" && file != "" {
			fileInfos := strings.Split(file, ".")

			// get extentions
			if len(fileInfos) > 1 {
				fileExt := fileInfos[1]

				// ensure these source code file endings
				if classifier.GetInstance().IsCodeExtention(fileExt) {

					regionDiff[file] = []string{}
				}
			}
		}
	}

	//split all the different regions
	var regions = strings.Split(diff, "diff --git")[1:]

	for _, region := range regions {

		//We begin by splitting on the beginning of double at characters, which gives us an array looking like this:
		// [file info, line info {double at characters} modified code]
		initialChunks := strings.Split(region, ":BUMPER_DELIMITER_START:@@")

		//if a binary file it doesn't display the lines modified (a.k.a the 'line info {double at characters} modified code' part)
		if len(initialChunks) == 1 {
			continue
		}

		// file info is the first 'chunk', followed by the line_info {double at characters} modified code
		fileInfo := initialChunks[0]
		fileInfoSplit := strings.Split(fileInfo, " ")
		//remove the 'a/ character'
		fileName := fileInfoSplit[1][2:]

		//it is possible there is a binary file being tracked or something we shouldn't care about
		if _, present := regionDiff[fileName]; fileName == "" || !present {
			continue
		}

		// Next, we must know the lines modified so that we can annotate. To do this, we must further split the chunks_initial.
		// Specifically, we must seperate the line info from the code info. The second part of the initial chunk looks like
		// -101,30, +202,33 {double at characters} code modified info. We can be pretty certain that the line info doesnt contain
		// any at characters, so we can safely split the first set of doule at characters seen to divide this info up.

		// Iterate through - as in one file we can multiple sections modified.
		for _, chunk := range initialChunks[1:] {

			// split only on the first occurance of the double at characters
			codeInfoChunk := strings.Split(strings.Replace(chunk, "@@", "___%UNIQUE%___", 1), "___%UNIQUE%___")

			//This now contains the -101,30 +102,30 part (info about the lines modified)
			lineInfo := codeInfoChunk[0]
			//This now contains the modified lines of code seperated by the delimiter we set
			codeInfo := codeInfoChunk[1]

			// As we only care about modified lines of code, we must ignore the +/additions as they do exist in previous versions
			// and thus, we cannot even annotate them (they were added in this commit). So, we only care about the start where it was
			// modified and we will have to study which lines where modified and keep track of them.

			// remove clutter -> we only care about what line the modificatin started, first index is just empty
			modLineInfo := strings.Split(lineInfo, " ")[1]
			// remove clutter -> first line contains info on the class and last line irrelevant
			modCodeInfo := strings.Split(strings.Replace(codeInfo, "\\n", "", -1), ":BUMPER_DELIMITER:")
			modCodeInfo = modCodeInfo[1 : len(modCodeInfo)-1]

			// make sure this is legitimate. expect modified line info to start with '-'
			if modLineInfo[0] != '-' {
				continue
			}

			//remove comma from mod_line_info as we only care about the start of the modification
			if strings.Index(modLineInfo, ",") != -1 {
				modLineInfo = modLineInfo[0:strings.Index(modLineInfo, ",")]
			}

			//remove the '-' in front of the line number by abs
			currentLine, _ := strconv.ParseFloat(modLineInfo, 64)
			currentLine = math.Abs(currentLine)

			//now only use the code line changes that MODIFIES (not adds) in the diff
			for _, section := range modCodeInfo {

				// this line modifies or deletes a line of code
				if strings.Index(section, ":BUMPER_DELIMITER_START:-") != -1 {

					regionDiff[fileName] = append(regionDiff[fileName], strconv.FormatFloat(currentLine, 'f', 0, 64))

					// we only increment modified lines of code because those lines did NOT exist
					// in the previous commit!
					currentLine++
				}
			}
		}
	}

	return regionDiff
}

func (git *CMD) annotate(regionChunks map[string][]string, commit Commit, repoDir string) map[string]struct{} {

	bugIntroducingChanges := make(map[string]struct{})

	for file, lines := range regionChunks {

		for _, line := range lines {

			if line != "0" {

				// files changed, this is used by the getLineNumbersChanged function
				blameArgs := []string{
					"cd",
					repoDir,
					"&&",
					"git",
					"blame", "-L" + line + ",+1",
					commit.CommitHash + "^",
					"-l",
					"--",
					"'" + file + "'",
				}

				cmd := exec.Command("bash", "-c", strings.Join(blameArgs, " "))
				cmd.Dir = repoDir

				var buggyChanges []byte
				var err error

				if buggyChanges, err = cmd.Output(); err != nil {
					log.Panic("There was an error running git blame command: ", err, string(buggyChanges))
					os.Exit(1)
				}

				//we need to git blame with the --follow option so that it follows renames in the file, and the '-l'
				// option gives us the complete commit hash. additionally, start looking at the commit's ancestor
				buggyChangesString := strings.Split(string(buggyChanges), " ")[0]

				if _, present := bugIntroducingChanges[buggyChangesString]; !present {
					bugIntroducingChanges[buggyChangesString] = struct{}{}
				}

			}
		}
	}

	return bugIntroducingChanges
}
