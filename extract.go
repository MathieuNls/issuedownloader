package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/mathieunls/deepchange-downloader/git"
)

func main() {

	gitCMD := git.New()

	gitCMD.Commits("D:\\ace", "")

	// basePath := pwd + "/src/github.com/BUMPER/IssueDownloader/data/netbeans/"
	// baseURL := "https://netbeans.org/bugzilla/show_bug.cgi?ctype=xml&id="
	// db, err := sql.Open("mysql", "root:root@tcp(192.168.0.112:3306)/taxo")

	// if err != nil {
	// 	fmt.Printf(err.Error())
	// 	return
	// }

	// inFile, _ := os.Open(pwd + "/src/github.com/BUMPER/IssueDownloader/data/netbeans.csv")
	// defer inFile.Close()
	// scanner := bufio.NewScanner(inFile)
	// scanner.Split(bufio.ScanLines)

	// for scanner.Scan() {
	// 	id := scanner.Text()
	// 	report, _ := bugzilla.New(basePath+id+".xml", baseURL+id)
	// 	report.WriteWordsToDisk(db, "bug_netbeans_")
	// }

	// inFile, _ = os.Open(pwd + "/src/github.com/BUMPER/IssueDownloader/data/apache.csv")
	// defer inFile.Close()
	// scanner = bufio.NewScanner(inFile)
	// scanner.Split(bufio.ScanLines)

	// for scanner.Scan() {
	// 	id := scanner.Text()
	// 	basePath := pwd + "/src/github.com/BUMPER/IssueDownloader/data/apache/"
	// 	baseURL := "https://issues.apache.org/jira/si/jira.issueviews:issue-xml/" + id + "/" + id + ".xml"
	// 	reportb, _ := jira.New(basePath+id+".xml", baseURL)
	// 	reportb.WriteWordsToDisk(db, "bug_apache_")
	// }

	// //We're done with mysql.
	// //All in memory from here
	// db.Close()

	//wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_", []string{"ambari"}, []string{"1", "2", "3", "4"}, []string{"3"}, 5, 10)

	// apacheProducts := []string{"ambari", "hbase", "cassandra", "hive", "flume"}
	// netbeansProducts := []string{"editor", "javaee", "cnd", "java", "platform"}

	// // wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_", []string{apacheProducts[0]}, []string{"1", "2", "3", "4"}, []string{"4"}, 10, 20)
	// // wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_", []string{apacheProducts[0]}, []string{"1", "2", "3", "4"}, []string{"3"}, 10, 20)
	// // // wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_", []string{apacheProducts[0]}, []string{"1", "2", "3", "4"}, []string{"4"}, 10, 20)

	// for i := 0; i < len(apacheProducts); i++ {
	// 	wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_", []string{apacheProducts[i]}, []string{"1", "2", "3", "4"}, []string{"3"}, 500, 500000, false)
	// 	//wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_", []string{apacheProducts[i]}, []string{"1", "2", "3", "4"}, []string{"4"}, 40, 120, false)
	// 	// wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_", []string{apacheProducts[i]}, []string{"1", "2", "3", "4"}, []string{"3", "4"}, 40, 120, false)

	// 	// wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_", []string{apacheProducts[i]}, []string{"1", "2", "3", "4"}, []string{"4"}, 40, 120)
	// }

	// for i := 0; i < len(netbeansProducts); i++ {
	// 	wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/netbeans.tf.wnet", "bug_netbeans_", []string{netbeansProducts[i]}, []string{"1", "2", "3", "4"}, []string{"3"}, 500, 500000, false)
	// 	//wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/netbeans.tf.wnet", "bug_netbeans_", []string{netbeansProducts[i]}, []string{"1", "2", "3", "4"}, []string{"4"}, 40, 120, false)
	// 	// wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/netbeans.tf.wnet", "bug_netbeans_", []string{netbeansProducts[i]}, []string{"1", "2", "3", "4"}, []string{"3", "4"}, 40, 120, false)
	// 	// wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/netbeans.tf.wnet", "bug_netbeans_", []string{netbeansProducts[i]}, []string{"1", "2", "3", "4"}, []string{"4"}, 40, 80)
	// }

	//wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/netbeans.tf.wnet", "bug_netbeans_")
	//wordnet.IdfMap(pwd+"/src/github.com/BUMPER/IssueDownloader/data/tf/apache.tf.wnet", "bug_apache_")

	/*
		COMMANDS

		- Train files  ls *.libsvm | while read file; do lines=$(cat $file | wc -l); train=$(($lines / 10 * 8)); head $file -n $train > $file.train;  done
		- Test files: ls *.libsvm | while read file; do lines=$(cat $file | wc -l); train=$(($lines / 10 * 8)); test=$(($lines - $train)); tail $file -n $test > $file.test;  done
		- Run tests ls  -- *test *train | sed -e 's/\.train.*$//' | sed -e 's/\.test*$//' | uniq | while read file; do echo $file >> r.txt; hectorcv --method svm --train $file.train --test $file.test --cv 5 | tail -n 1 >> r.txt; done
		- cleanup rm *test *train *libsvm *xml *txt
	*/

}
