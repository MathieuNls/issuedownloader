package git

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"io/ioutil"

	"sync"

	classifier "github.com/mathieunls/deepchange-downloader/classifiers"
	"github.com/mathieunls/deepchange-downloader/jira"
	"github.com/mathieunls/deepchange-downloader/persistence"
	"github.com/mathieunls/deepchange-downloader/pogo"
)

// CMD is an abstraction of diverse git commands
type CMD struct {
	logformat         string
	cloneCMD          string
	pullCMD           string
	resetCMD          string
	cleanCMD          string
	headCommitHashCMD string
	Threads           int
	ReportLinker      pogo.ReportLinker
	DBAdaptor         persistence.DBAdaptor
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
	g.Threads = 12
	g.ReportLinker = nil
	g.DBAdaptor = nil
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
	commit *pogo.Commit) {

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
func (git *CMD) Commits(
	repoDir string,
	repoName string,
	lastIngestedCommit string,
	workingDir string,
	repositoryID int) ([]*pogo.Commit, []*pogo.Commit) {

	var cmdOut []byte
	var err error

	if repoDir != workingDir {
		fmt.Println("workingDir + repoName", workingDir+repoName)
		if _, err = os.Stat(workingDir + repoName); err != nil {
			git.cloneRepo(repoDir+repoName, workingDir+repoName, true)
		}
	}

	// Log file already exists
	if _, err = os.Stat(workingDir + "logs/" + repoName + ".log"); err == nil {

		fmt.Println("Found file", workingDir+"/logs/"+repoName+".log")
		cmdOut, err = ioutil.ReadFile(workingDir + "/logs/" + repoName + ".log")

		if err != nil {
			panic(err)
		}
	} else {
		//Run git log
		cmdArgs := []string{"log", "--numstat", "--reverse", git.logformat}

		if lastIngestedCommit != "" {
			cmdArgs = append(cmdArgs, fmt.Sprintf("%s..HEAD", lastIngestedCommit))
		}

		cmdArgs = append(cmdArgs, "> "+workingDir+"/logs/"+repoName+".log")

		cmd := exec.Command("git", cmdArgs...)
		cmd.Dir = workingDir

		if cmdOut, err = cmd.Output(); err != nil {
			log.Panic("There was an error running git log command: ", err)
			os.Exit(1)
		}

		//Restart this we the outputed log file
		git.Commits(repoDir, repoName, lastIngestedCommit, workingDir, repositoryID)

	}

	commitFiles := make(map[string]commitFile)
	devExp := make(map[string]devExperiences)
	commitList := strings.Split(string(cmdOut), "BUMPER_STARTPRETTY")

	commits := []*pogo.Commit{}
	trueCorrectiveCommits := []*pogo.Commit{}
	correctiveCommits := []*pogo.Commit{}
	totalFixReports := 0
	syncEnable := false

	if strings.Compare("", lastIngestedCommit) == 0 {
		syncEnable = true
	}

	for index := 1; index < len(commitList); index++ {

		prettyCommitSplit := strings.Split(commitList[index], "BUMPER_STOPPRETTY")
		prettyCommit := prettyCommitSplit[0]
		statsCommit := prettyCommitSplit[1]

		prettyCommitDetails := strings.Split(prettyCommit, " BUMPER_DELIMITER2")

		commit := pogo.NewCommit(
			strings.Split(prettyCommitDetails[0], " "),
			strings.Trim(prettyCommitDetails[1], " "),
			strings.Trim(prettyCommitDetails[2], " "),
			strings.Trim(prettyCommitDetails[3], " "),
			strings.Trim(prettyCommitDetails[4], " "),
			strings.Trim(prettyCommitDetails[5], " "),
			strings.Trim(prettyCommitDetails[6], " "),
			`@fix(ed\()?( )?([a-zA-Z0-9]+-[0-9]+)`,
			`@review\(([a-z,]+)\)`,
			true,
			repositoryID)

		stats := strings.Split(statsCommit, "\n")
		git.commitStats(
			stats,
			commitFiles,
			devExp,
			commit.AuthorEmail,
			commit.AuthorDateUnixTimestamp,
			commit)

		if _, present := commit.Classification["corrective"]; present {
			if commit.Classification["corrective"] == 100.0 && len(commit.Classification) == 1 {
				trueCorrectiveCommits = append(trueCorrectiveCommits, commit)
			} else if commit.Classification["corrective"] > 0.0 {
				correctiveCommits = append(correctiveCommits, commit)
			}
		}

		if git.DBAdaptor != nil && syncEnable {

			git.DBAdaptor.SyncCommit(commit)
		} else {
			fmt.Println("Skipping", commit.CommitHash)
		}

		if strings.Compare(commit.CommitHash, lastIngestedCommit) == 0 {
			syncEnable = true
		}

		totalFixReports += len(commit.FixReportIDs)
		commits = append(commits, commit)
	}

	fmt.Println("Commits:", len(commits))
	fmt.Println("Pure Corrective Commits:", len(trueCorrectiveCommits))
	fmt.Println("Corrective Commits:", len(correctiveCommits))
	fmt.Println("Reports closed:", totalFixReports)

	return commits, trueCorrectiveCommits
}

//clone a repo
func (git *CMD) cloneRepo(from string, to string, bare bool) {

	fmt.Println("Copying from", from, "to", to, "with bare =", bare)

	cmdArgs := []string{"git", "clone"}

	if bare {
		cmdArgs = append(cmdArgs, "--bare")
	}

	/*
		core.preloadindex
		Enable parallel index preload for operations like git diff
		This can speed up operations like git diff and git status
		especially on filesystems like NFS that have weak caching semantics
		and thus relatively high IO latencies. With this set to true,
		git will do the index comparison to the filesystem data in parallel,
		allowing overlapping IO's.
	*/
	cmdArgs = append(cmdArgs, from, to, "&& cd", from, "&& git config core.preloadindex true")

	cmd := exec.Command("bash", "-c", strings.Join(cmdArgs, " "))
	_, err := cmd.CombinedOutput()

	if err != nil {
		panic(err)
	}

	fmt.Println("Copy from", from, "to", to, " done.")
}

type commitChan struct {
	Commit *pogo.Commit
	ID     int
}

//LinkCorrectiveCommits tries to link fault commits with their fixes
func (git *CMD) LinkCorrectiveCommits(
	correctiveCommits []*pogo.Commit,
	allCommits []*pogo.Commit, repoDir string,
	logDir string, repoID int) {

	//Parallel stuff
	jobs := make(chan commitChan, 15000) //len(correctiveCommits))
	results := make(chan map[string][]string)
	wg := sync.WaitGroup{}
	wg.Add(git.Threads)

	//Create git.Threads copies of the repo
	for w := 0; w < git.Threads; w++ {
		go func(workerId int) {

			git.cloneRepo(repoDir, repoDir+"-bare-"+strconv.Itoa(workerId), true)
			wg.Done()
		}(w)
	}
	wg.Wait()

	//Contains the result of the linking operations
	//bug in hash -> introducted by hashes
	linkedCommits := make(map[string][]string)

	//create worker to operate parrallel blames & annotates
	for w := 0; w < git.Threads; w++ {
		go git.linkerWorker(jobs, repoDir+"-bare-"+strconv.Itoa(w), results, w, len(correctiveCommits), logDir, repoID)
		fmt.Println("creating worker", w)
	}

	//Feed our bug fixes to the workers
	for i := 0; i < 15000; i++ { //len(correctiveCommits); i++ {
		jobs <- commitChan{correctiveCommits[i], i}
	}
	close(jobs)

	for i := 0; i < 15000; i++ { //len(correctiveCommits); i++ {
		fmt.Println("received", i)

		for k, v := range <-results {
			//response for a timeout is null
			if v != nil {
				linkedCommits[k] = append(linkedCommits[k], v...)
			}
		}
	}

	for _, commit := range allCommits {
		if _, present := linkedCommits[commit.CommitHash]; present {
			commit.ContainsBug = true
			commit.FixHashes = linkedCommits[commit.CommitHash]
			if git.DBAdaptor != nil {
				git.DBAdaptor.IsBuggy(commit, repoID)
			}
		}
	}

	//Delete the repos, no need to wait for complete deletion here
	for w := 0; w < git.Threads; w++ {
		go func(workerId int) {

			cmd := exec.Command("rm", "-r", repoDir+"-bare-"+strconv.Itoa(workerId))
			_, err := cmd.CombinedOutput()

			if err != nil {
				panic(err)
			}
			fmt.Println("deleting repo", repoDir+"-bare-"+strconv.Itoa(workerId))
			wg.Done()
		}(w)
	}

}

//linkerWorker is a worker that performs git blame/annotate and
//updated the linkedCommit map
func (git *CMD) linkerWorker(
	correctiveCommits <-chan commitChan,
	repoDir string,
	results chan map[string][]string,
	id int,
	total int,
	logDir string,
	repoID int) {

	for correctiveCommit := range correctiveCommits {

		wg := sync.WaitGroup{}
		wg.Add(2)

		linkedCommits := make(map[string][]string)

		//First thread to blame the corrective commit
		go func(localWg *sync.WaitGroup, commit *pogo.Commit) {

			regionChunks := git.getModifiedRegions(commit, repoDir, logDir)
			bugIntroducingChanges := git.annotate(regionChunks, commit, repoDir, logDir)

			for buggyCommit := range bugIntroducingChanges {

				linkedCommits[buggyCommit] = append(linkedCommits[buggyCommit], commit.CommitHash)
			}
			commit.Linked = true
			if git.DBAdaptor != nil {
				git.DBAdaptor.IsLinked(commit, repoID)
			}
			localWg.Done()
		}(&wg, correctiveCommit.Commit)

		//Second thread to fetch fixed reports
		go func(localWg *sync.WaitGroup, commit *pogo.Commit) {

			//Do we have a report linker ?
			if git.ReportLinker != nil {

				var pogoReport pogo.Report
				var err error

				//Ids are extracted at commit instantiation
				for _, reportID := range commit.FixReportIDs {
					fmt.Println("fetching report", git.ReportLinker.DBName()+"_"+reportID)
					//Do we have that report in cache ?
					if report := persistence.GetCacheInstance().
						Fetch("report", git.ReportLinker.DBName()+"_"+strings.Replace(reportID, "ACE-", "", 1)); report != nil {
						fmt.Println("Cache hit report")
						pogoReport = &jira.Report{ReportAttributes: report.(pogo.ReportAttributes)}
					} else {
						pogoReport, err = git.ReportLinker.Fetch(reportID)
					}

					if err != nil {
						fmt.Println(err.Error(), correctiveCommit, reportID)
					} else {
						commit.FixReports = append(commit.FixReports, pogoReport)
					}

				}
			}

			//Do we have a db adapotor sync reports ?
			if git.DBAdaptor != nil {
				git.DBAdaptor.SyncReports(commit.FixReports, commit.RepositoryID, commit.CommitHash)
			}

			localWg.Done()
		}(&wg, correctiveCommit.Commit)

		if waitTimeout(&wg, time.Hour) {
			fmt.Println("worker", id, "/", git.Threads, "timed out", correctiveCommit.ID, "/", total, float64(correctiveCommit.ID)/float64(total)*100, "%")
			results <- nil
		} else {
			results <- linkedCommits
			fmt.Println("worker", id, "/", git.Threads, "finished job", correctiveCommit.ID, "/", total, float64(correctiveCommit.ID)/float64(total)*100, "%")
		}
	}
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

//  getModifiedRegions returns the list of regions that were modified/deleted between this commit and its ancester.
// a region is simply the file and the loc in it that were modified.
func (git *CMD) getModifiedRegions(commit *pogo.Commit, repoDir string, logDir string) map[string][]string {

	var diff []byte
	var filesModifiedOUT []byte
	var err error
	var cmdArgs []string

	diffID := "diff_" + commit.CommitHash + "^" + commit.CommitHash
	filesModifiedDiffID := "file_modified_diff_" + commit.CommitHash + "^" + commit.CommitHash

	if cachedDiffed := persistence.GetCacheInstance().
		Fetch("diff", diffID); cachedDiffed != nil {
		diff = cachedDiffed.([]byte)
	} else {
		cmdArgs = []string{
			"git",
			"diff",
			commit.CommitHash + "^",
			commit.CommitHash,
			"--unified=0",
			" | while",
			"read; do echo \":BUMPER_DELIMITER_START:$REPLY:BUMPER_DELIMITER:\"; done",
			"> " + logDir + diffID,
			"&& cat " + logDir + diffID,
		}

		cmd := exec.Command("bash", "-c", strings.Join(cmdArgs, " "))
		cmd.Dir = repoDir

		if diff, err = cmd.CombinedOutput(); err != nil {
			log.Panic("There was an error running git diff command: ", err,
				"--- bash ", "-c ", strings.Join(cmdArgs, " "), "at", repoDir, string(diff))

			//write an empty file so we don't wait here ever again
			exec.Command("bash", "-c", "touch "+logDir+diffID).Output()
			persistence.GetCacheInstance().
				Put("diff", diffID, []byte{})
		} else {

			persistence.GetCacheInstance().
				Put("diff", diffID, diff)
		}
	}

	if cachedDiffed := persistence.GetCacheInstance().
		Fetch("file_modified_diff", filesModifiedDiffID); cachedDiffed != nil {
		filesModifiedOUT = cachedDiffed.([]byte)
	} else {
		nameOnlyArgs := []string{
			"git",
			"diff",
			commit.CommitHash + "^",
			commit.CommitHash,
			"--name-only",
			"> " + logDir + filesModifiedDiffID,
			"&& cat " + logDir + filesModifiedDiffID,
		}

		cmd := exec.Command("bash", "-c", strings.Join(nameOnlyArgs, " "))
		cmd.Dir = repoDir

		// get the files modified -> use this to validate if we have arrived at a new file
		// when grepping for the specific lines changed.
		if filesModifiedOUT, err = cmd.CombinedOutput(); err != nil {
			log.Println("There was an error running git diff command: ", err,
				"--- git ", strings.Join(nameOnlyArgs, " "), " at", repoDir, "previous command was",
				"--- bash ", "-c ", strings.Join(cmdArgs, " "), " at", repoDir, string(filesModifiedOUT))

			//write an empty file so we don't wait here ever again
			exec.Command("bash", "-c", "touch "+logDir+filesModifiedDiffID).Output()
			persistence.GetCacheInstance().
				Put("file_modified_diff", filesModifiedDiffID, []byte{})
		} else {

			persistence.GetCacheInstance().
				Put("file_modified_diff", filesModifiedDiffID, filesModifiedOUT)
		}

	}

	filesModified := strings.Split(strings.Replace(string(filesModifiedOUT), "b'", "", -1), "\n")

	if err != nil {
		return make(map[string][]string)
	}

	return git.extractRegions(string(diff), filesModified)
}

//  extractRegions returns a dict of file -> list of line numbers modified. helper function for getModifiedRegions
//  git diff doesn't provide a clean way of simply getting the specific lines that were modified, so we are doing so here
//  manually. A possible refactor in the future may be to use an external diff tool, so that this implementation
//  wouldn't be scm (git) specific
//  if a file was merely deleted, then there was no chunk or region changed but we do capture the file.
//  however, we do not assume this is a location of a buy
//  modified means modified or deleted -- not added! We assume are lines of code modified is the location of a bug.
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

			if len(modCodeInfo) > 2 {
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
	}

	return regionDiff
}

// annotate tracks down the origin of the deleted/modified loc in the regions dict using
// the git annotate (now called git blame) feature of git and a list of commit
// hashes of the most recent revision in which the line identified by the regions
// was modified. these discovered commits are identified as bug-introducing changes.
// git blame command is set up to start looking back starting from the commit BEFORE the
// commit that was passed in. this is because a bug MUST have occured prior to this commit.
func (git *CMD) annotate(regionChunks map[string][]string, commit *pogo.Commit, repoDir string, logDir string) map[string]struct{} {

	bugIntroducingChanges := make(map[string]struct{})

	for file, lines := range regionChunks {

		for _, line := range lines {

			if line != "0" {

				var buggyChanges []byte
				var err error
				blameID := "blame_" + line + "_" + commit.CommitHash + "_" + strings.Replace(file, "/", "--", -1)

				if cachedBlame := persistence.GetCacheInstance().
					Fetch("blame", blameID); cachedBlame != nil {
					buggyChanges = cachedBlame.([]byte)
				} else {

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
						"> " + logDir + blameID,
						"&& cat " + logDir + blameID,
					}

					cmd := exec.Command("bash", "-c", strings.Join(blameArgs, " "))
					cmd.Dir = repoDir

					if buggyChanges, err = cmd.CombinedOutput(); err != nil {
						fmt.Println("There was an error running git blame command: ", "bash -c", strings.Join(blameArgs, " "), err.Error())
						//write an empty file so we don't wait here ever again
						exec.Command("bash", "-c", "touch "+logDir+blameID).Output()
						persistence.GetCacheInstance().
							Put("blame", blameID, []byte{})
					} else {
						persistence.GetCacheInstance().
							Put("blame", blameID, buggyChanges)
					}

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
