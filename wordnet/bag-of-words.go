package wordnet

import (
	"encoding/xml"
	"errors"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"fmt"

	"github.com/fluhus/gostuff/nlp"
	"github.com/kennygrant/sanitize"
)

//ExtractUniqWords returns an weighted hashmap without stopword
func ExtractUniqWords(text string) map[string]float32 {

	words := strings.Fields(strings.ToLower(sanitize.HTML(removePunctuation(text))))
	m := make(map[string]int)

	for index := 0; index < len(words); index++ {

		m = updateMap(m, nlp.Stem(words[index]))

	}

	return tfMap(m, len(words))
}

//ExtractUniqGrams retuns a weighted hashmap of grams without stopwords
func ExtractUniqGrams(text string, grams int) map[string]float32{
//	fmt.Println("dbPrefix", text)	
	words := strings.Fields(strings.ToLower(sanitize.HTML(removePunctuation(text))))
//	fmt.Println("dbPrefix", words)
	m := make(map[string]int)
	
	for index := 0; index < len(words) - grams; index++{

		word := ""

		for gram := 0; gram < grams; gram++{
			word = word + " " + nlp.Stem(words[index + gram])
		}

		m = updateMap(m, word)
	}

	return tfMap(m, len(words))
}

func tfMap(m map[string]int, count int) map[string]float32 {

	tfMap := make(map[string]float32)
	// Write the words to file
	for key, value := range m {
		tfMap[key] = float32(value) / float32(count)
	}

	return tfMap
}

func IdfMap(filePath string, dbPrefix string, products []string,
	investiagteTypes []string, groupingTypes []string,
	wordsPerReport int, wordsForAllReport int, balanceSet bool) error {

	fmt.Println("dbPrefix", dbPrefix)
	fmt.Println("product", products)
	fmt.Println("investiagteType", investiagteTypes)
	fmt.Println("groupingTypes", groupingTypes)

	type wordTf struct {
		Text  string  `xml:",chardata"`
		Tf    float64 `xml:"tf,attr"`
		TfIdf float64
	}

	type termFrequencyReport struct {
		ID         string   `xml:"id,attr"`
		Product    string   `xml:"product,attr"`
		Words      []wordTf `xml:"word"`
		ReportType string   `xml:"type,attr"`
	}

	type significantWord struct {
		sumTfIdf float64
		amount   float64
		word     string
		avf      float64
	}

	type vector struct {
		reportType string
		svnClass   string
		vector     string
	}

	var reports []termFrequencyReport

	//XML File opening
	xmlFile, err := os.Open(filePath)
	if err != nil {
		return errors.New("Error opening file:" + filePath)
	}
	defer xmlFile.Close()

	//Parse the xml file
	decoder := xml.NewDecoder(xmlFile)
	var inElement string
	for {
		report := termFrequencyReport{}
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			break
		}

		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			// If we just read a StartElement token
			inElement = se.Name.Local
			// ...and its name is "report"
			if inElement == "report" {
				decoder.DecodeElement(&report, &se)

				// Only append report that belongs the the product
				// under analyse
				if contains(products, report.Product) {

					// if contains(investiagteTypes, report.ReportType) {

					reports = append(reports, report)
					// }
				}

			}
		default:
		}

	}

	trainingLen := int(len(reports) / 100 * 70)

	//On the fetched reports, count the occcurences of
	//each words
	count := 0
	trainingWords := make(map[string]int)
	testingWords := make(map[string]int)
	for i := 0; i < len(reports); i++ {
		count = count + 1
		for j := 0; j < len(reports[i].Words); j++ {
			if i < trainingLen {
				trainingWords = updateMap(trainingWords, reports[i].Words[j].Text)
			}
			testingWords = updateMap(testingWords, reports[i].Words[j].Text)
		}

	}

	//This map will be used to compute an average tfidf of the top
	//n words in each reports
	significantWords := make(map[string]significantWord)

	// Create the file for storing transformed report
	out, err := os.Create(filePath + strings.Join(products, "-") + "-" + strings.Join(investiagteTypes, "-") + "-" + strings.Join(groupingTypes, "") + ".tfidf.wnet.xml")
	if err != nil {
		return err
	}
	defer out.Close()

	for i := 0; i < len(reports); i++ {

		//First line of a report
		out.WriteString("<report id=\"" + reports[i].ID + "\" type=\"" + reports[i].ReportType + "\" product=\"" + reports[i].Product + "\">\n")

		for j := 0; j < len(reports[i].Words); j++ {

			//Bunch of math transformed to strings for xml writes
			tf := float64(reports[i].Words[j].Tf)
			var idf float64
			if i < trainingLen {
				idf = math.Log(float64(count) / float64(trainingWords[reports[i].Words[j].Text]))
			} else {
				idf = math.Log(float64(count) / float64(testingWords[reports[i].Words[j].Text]))
			}
			tfidf := tf * idf
			tfStr := strconv.FormatFloat(float64(tf), 'f', 5, 64)
			idfStr := strconv.FormatFloat(float64(idf), 'f', 5, 64)
			tfidfStr := strconv.FormatFloat(float64(tfidf), 'f', 5, 64)

			//Store the tfidf, computed here, for later use
			reports[i].Words[j].TfIdf = tfidf
			// fmt.Println("\t<word tf=\"" + tfStr + "\" idf=\"" + idfStr + "\" tfidf=\"" + tfidfStr + "\">" + reports[i].Words[j].Text + "</word>\n")

			//Compute and write the xml string for a word inside a report
			str := "\t<word tf=\"" + tfStr + "\" idf=\"" + idfStr + "\" tfidf=\"" + tfidfStr + "\">" + reports[i].Words[j].Text + "</word>\n"
			out.WriteString(str)
		}

		//Closing tag for the report
		out.WriteString("</report>\n")

		if i < trainingLen {
			//If the report belongs to the type under analyse,
			//we sort each word by their tfidf value then we store
			//wordPerReport most significant into the significantWords map
			//
			//We don't filter reports that are not investiagteType
			//at mapping time as we need them for training/testing
			//our svm models
			if contains(investiagteTypes, reports[i].ReportType) {
				for j := 0; j < len(reports[i].Words); j++ {
					for k := 0; k < len(reports[i].Words)-1; k++ {
						if reports[i].Words[k].TfIdf < reports[i].Words[k+1].TfIdf {
							tmp := reports[i].Words[k+1]
							reports[i].Words[k+1] = reports[i].Words[k]
							reports[i].Words[k] = tmp
						}
					}
				}
			}

			// fmt.Println(reports[i].Words)

			for l := 0; l < len(reports[i].Words) && l < wordsPerReport; l++ {

				// Go bug, can't assess field of a struct inside a map
				// https://github.com/golang/go/issues/3117
				word := significantWords[reports[i].Words[l].Text]
				word.amount++
				word.sumTfIdf += reports[i].Words[l].TfIdf
				word.word = reports[i].Words[l].Text
				significantWords[reports[i].Words[l].Text] = word
			}
		}

	}

	//Here, we are actually beginning the vector preparation
	//These two variables will stores significantWords extracted
	//earlier into struct and simple string arrays
	//
	//The struct array is used to compute the avg tfidf and
	//the string array to build the libsvm vector
	var significantWordsArray []significantWord
	var significantWordsString []string

	//Compute average from the map and store to the array
	for _, v := range significantWords {
		v.avf = v.sumTfIdf / v.amount
		significantWordsArray = append(significantWordsArray, v)
	}

	//Bubble sort on avf (Average tfidf)
	for i := 0; i < len(significantWordsArray); i++ {
		for j := 0; j < len(significantWordsArray)-1; j++ {
			if significantWordsArray[j].amount < significantWordsArray[j+1].amount {
				tmp := significantWordsArray[j+1]
				significantWordsArray[j+1] = significantWordsArray[j]
				significantWordsArray[j] = tmp
			}
		}
	}

	//Extract the top wordsForAllReport words accross all reports of investiagteType tyoe
	for i := 0; i < len(significantWordsArray) && i < wordsForAllReport; i++ {
		significantWordsString = append(significantWordsString, significantWordsArray[i].word)
	}

	//Maps of vectors that will contain all the vectors created
	vectors := make(map[string][]vector)

	//Create the csv header line
	csvHeader := "classification_taxo,"
	for k := 0; k < len(significantWordsString); k++ {
		csvHeader = csvHeader + significantWordsString[k] + ","
	}
	vectors["csv"] = append(vectors["csv"], vector{reportType: "0", svnClass: "", vector: csvHeader})

	//Going through all the reports again to generate the vectors
	//We can't do that in the first pass as we need the average of tfidf
	//leading to the generation of significantWordsString
	for i := 0; i < len(reports); i++ {

		//We store two vectors, one weighted by tfidf and one binary (i.e. present or not)
		wordVector := ""
		csvVector := ""

		// binaryVector := ""
		qualityIndex := 0

		for k := 0; k < len(significantWordsString); k++ {

			csvValue := "0"

			for j := 0; j < len(reports[i].Words); j++ {

				if reports[i].Words[j].Text == significantWordsString[k] {
					wordVector = wordVector + " " + strconv.Itoa(k) + ":" + strconv.FormatFloat(float64(reports[i].Words[j].TfIdf), 'f', 5, 64)
					// binaryVector = binaryVector + " " + strconv.Itoa(k) + ":1"
					qualityIndex++
					csvValue = strconv.FormatFloat(float64(reports[i].Words[j].TfIdf), 'f', 5, 64)
				}
			}

			csvVector = csvVector + csvValue + ","

		}

		prefix := reports[i].ReportType

		if len(groupingTypes) != 0 {

			if contains(groupingTypes, reports[i].ReportType) {
				prefix = "1"
			} else {
				prefix = "-1"
			}
		}

		vectors["weighted"] = append(vectors["weighted"], vector{reportType: reports[i].ReportType, svnClass: prefix, vector: prefix + "  " + wordVector})
		vectors["csv"] = append(vectors["csv"], vector{reportType: reports[i].ReportType, svnClass: prefix, vector: prefix + "," + csvVector})
		// vectors["binary"] = append(vectors["binary"], vector{reportType: reports[i].ReportType, svnClass: prefix, vector: prefix + "  " + binaryVector})

	}

	//Iterates over all the vectors
	//Here's we apply create testing, validating and training sets
	//training sets are oversampled in order to create a normal distribution
	for k, v := range vectors {

		classOne := 0
		classTwo := 0
		var zeror string

		var trainingSet []vector
		var testingSet []vector

		for i := 0; i < trainingLen; i++ {
			trainingSet = append(trainingSet, v[i])
		}

		for i := trainingLen; i < len(v); i++ {
			testingSet = append(testingSet, v[i])
		}

		//sort the trainingSet
		//This will be usefull for oversampling
		for i := 0; i < len(trainingSet); i++ {
			for j := 0; j < len(trainingSet)-1; j++ {
				if trainingSet[j].svnClass > trainingSet[j+1].svnClass {
					tmp := trainingSet[j+1]
					trainingSet[j+1] = trainingSet[j]
					trainingSet[j] = tmp
				}
			}
		}

		//Compute the distribution in the training set
		for i := 0; i < len(trainingSet); i++ {

			if v[i].svnClass == "-1" {
				classOne++
			} else {
				classTwo++
			}
		}

		if classOne > classTwo {
			zeror = "-1"
		} else {
			zeror = "1"
		}

		if balanceSet {
			balancedTrainingSet := trainingSet
			//Balance the set by oversampling one of two classes
			if classOne > classTwo {
				factor := float64(classOne) / float64(classTwo)
				for index := 0; index < int(factor-1); index++ {
					balancedTrainingSet = append(balancedTrainingSet, trainingSet[classOne:len(trainingSet)-1]...)
				}

				end := classOne + (classOne - classTwo*int(factor))

				balancedTrainingSet = append(balancedTrainingSet, trainingSet[classOne:end]...)

			} else if classTwo > classOne {
				factor := float64(classTwo) / float64(classOne)

				for index := 0; index < int(factor)-1; index++ {
					balancedTrainingSet = append(balancedTrainingSet, trainingSet[0:classOne]...)
				}

				end := float64(classOne) * (factor - 1.0)

				balancedTrainingSet = append(balancedTrainingSet, trainingSet[0:int(end)]...)
			}

			//sort the balanced training set for easy visual verification
			for i := 0; i < len(balancedTrainingSet); i++ {
				for j := 0; j < len(balancedTrainingSet)-1; j++ {
					if balancedTrainingSet[j].svnClass > balancedTrainingSet[j+1].svnClass {
						tmp := balancedTrainingSet[j+1]
						balancedTrainingSet[j+1] = balancedTrainingSet[j]
						balancedTrainingSet[j] = tmp
					}
				}
			}
			balancedTraining, _ := os.Create(filePath + strings.Join(products, "-") + "-" + strings.Join(investiagteTypes, "") + "-" + strings.Join(groupingTypes, "") + "-" + k + ".balancedtraining.libsvm")

			defer balancedTraining.Close()
			for i := 0; i < len(balancedTrainingSet); i++ {
				balancedTraining.WriteString(balancedTrainingSet[i].vector + "\n")
			}

		}

		training, _ := os.Create(filePath + strings.Join(products, "-") + "-" + strings.Join(investiagteTypes, "") + "-" + strings.Join(groupingTypes, "") + "-" + k + ".training.libsvm")
		testing, _ := os.Create(filePath + strings.Join(products, "-") + "-" + strings.Join(investiagteTypes, "") + "-" + strings.Join(groupingTypes, "") + "-" + k + ".testing.libsvm")
		all, _ := os.Create(filePath + strings.Join(products, "-") + "-" + strings.Join(investiagteTypes, "") + "-" + strings.Join(groupingTypes, "") + "-" + k + ".libsvm")
		zerorW, _ := os.Create(filePath + strings.Join(products, "-") + "-" + strings.Join(investiagteTypes, "") + "-" + strings.Join(groupingTypes, "") + "-" + k + ".zeror")
		random, _ := os.Create(filePath + strings.Join(products, "-") + "-" + strings.Join(investiagteTypes, "") + "-" + strings.Join(groupingTypes, "") + "-" + k + ".random")

		defer training.Close()
		defer testing.Close()
		defer all.Close()
		defer zerorW.Close()
		defer random.Close()

		for i := 0; i < len(trainingSet); i++ {
			training.WriteString(trainingSet[i].vector + "\n")
		}

		if k == "csv" {
			testing.WriteString(csvHeader + "\n")
		}

		for i := 0; i < len(testingSet); i++ {
			testing.WriteString(testingSet[i].vector + "\n")
			zerorW.WriteString(zeror + "\n")

			if rand.Intn(100) >= 50 {
				random.WriteString("1 " + testingSet[i].svnClass + "\n")
			} else {
				random.WriteString("-1 " + testingSet[i].svnClass + "\n")
			}
		}

		for i := 0; i < len(v); i++ {
			all.WriteString(v[i].vector + "\n")
		}

	}

	return nil
}

func contains(stringSlice []string, searchString string) bool {
	for _, value := range stringSlice {
		if strings.EqualFold(value, searchString) {
			return true
		}
	}
	return false
}

func updateMap(m map[string]int, word string) map[string]int {
	if !stopwords.Has(word) {
		if m[word] == 0 {
			m[word] = 1

		} else {
			m[word] = m[word] + 1
		}
	}
	return m
}

// Removes all punctuations
func removePunctuation(text string) string {

	reg, _ := regexp.Compile("[^A-Za-z]+")
	text = strings.ToLower(reg.ReplaceAllString(text, " "))

	return text
}
