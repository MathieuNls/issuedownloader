package classifier

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

type classifierSingleton struct {
	categories map[string][]string
}

var instance *classifierSingleton
var once sync.Once

func GetInstance() *classifierSingleton {
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
			r := csv.NewReader(csvFile)
			record, _ := r.Read()
			instance.categories[strings.Replace(file.Name(), ".csv", "", 1)] = record
		}
	})
	return instance
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
