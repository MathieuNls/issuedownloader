package classifier

import (
	"bufio"
	"encoding/csv"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

type classifierSingleton struct {
	categories     map[string][]string
	codeExtentions map[string]struct{}
}

var instance *classifierSingleton
var once sync.Once

func GetInstance() *classifierSingleton {

	if instance == nil {
		once.Do(func() {
			instance = &classifierSingleton{}

			instance.categories = make(map[string][]string)

			pwd, _ := os.Getwd()
			files, err := ioutil.ReadDir(pwd + "/classifiers/categories/")
			if err != nil {
				log.Fatal(err)
			}

			for _, file := range files {

				csvFile, _ := os.Open(pwd + "/classifiers/categories/" + file.Name())
				defer csvFile.Close()
				r := csv.NewReader(csvFile)
				record, _ := r.Read()
				instance.categories[strings.Replace(file.Name(), ".csv", "", 1)] = record
			}

			codeExtFile, _ := os.Open(pwd + "/classifiers/files_extensions/extentions.txt")
			defer codeExtFile.Close()

			instance.codeExtentions = make(map[string]struct{})
			scanner := bufio.NewScanner(codeExtFile)
			for scanner.Scan() {
				instance.codeExtentions[strings.ToLower(scanner.Text())] = struct{}{}
			}

		})
	}

	return instance
}

func (s *classifierSingleton) IsCodeExtention(ext string) bool {
	if _, present := s.codeExtentions[strings.ToLower(ext)]; present {
		return true
	}
	return false
}

func (s *classifierSingleton) Categorize(text string) []string {

	cats := []string{}

	for key, values := range s.categories {

		words := strings.Fields(text)

		if contains(words, values) {
			cats = append(cats, key)
		}

	}

	return cats
}

func contains(words []string, values []string) bool {
	for i := 0; i < len(words); i++ {

		for j := 0; j < len(values); j++ {

			if strings.Index(strings.ToLower(words[i]),
				strings.ToLower(values[j])) != -1 {

				return true
			}
		}
	}

	return false
}
