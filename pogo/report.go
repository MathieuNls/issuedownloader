package pogo

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mathieunls/deepchange-downloader/wordnet"
)

//Report represents a bug report
type Report interface {
	String() string
	DownloadFile() (err error)
	WriteWordsToDisk() (err error)
	AllText() string
}

type GenericReport struct {
	ID       string
	Product  string
	FilePath string
	Url      string
	AllText  func() string
	words    map[string]float32
}

func (report *GenericReport) Words() map[string]float32 {

	if report.words == nil {
		report.words = wordnet.ExtractUniqWords(report.AllText())
	}

	return report.words
}

// ParseXML parses a xml
func (report *GenericReport) ParseXML(token string, realReport interface{}) error {

	xmlFile, err := os.Open(report.FilePath)

	if err != nil {
		return errors.New("Error opening file:" + report.FilePath)
	}
	defer xmlFile.Close()

	decoder := xml.NewDecoder(xmlFile)

	var inElement string
	for {
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
			// ...and its name is token
			if inElement == token {
				decoder.DecodeElement(&realReport, &se)

			}
		default:
		}

	}

	return nil
}

//WriteNGramsToDisk to disk
func (report *GenericReport) WriteNGramsToDisk(db *sql.DB, dbPrefix string, grams int) (err error) {

	//Fetches bug type
	reportType := report.getType(db, dbPrefix)

	// Create the file
	out, err := os.Create(report.FilePath + "." + strconv.Itoa(grams) + "grams.wnet.xml")
	if err != nil {
		return err
	}
	defer out.Close()

	out.WriteString("<report id=\"" + report.ID + "\" product=\"" + report.Product + "\" type=\"" + reportType + "\">\n")
	// Write the words to file
	for key, value := range wordnet.ExtractUniqGrams(report.AllText(), grams) {

		out.WriteString("\t<word tf=\"" + strconv.FormatFloat(float64(value), 'f', 5, 64) + "\">" + key + "</word>\n")
	}
	out.WriteString("</report>\n")

	fmt.Println(report.ID + " grams saved")

	return nil
}

func (report *GenericReport) WriteWordsToDisk(db *sql.DB, dbPrefix string) (err error) {

	if report.words == nil {
		report.Words()
	}

	reportType := report.getType(db, dbPrefix)

	// Create the file
	out, err := os.Create(report.FilePath + ".wnet.xml")
	if err != nil {
		return err
	}
	defer out.Close()

	out.WriteString("<report id=\"" + report.ID + "\" product=\"" + report.Product + "\" type=\"" + reportType + "\">\n")
	// Write the words to file
	for key, value := range report.words {

		out.WriteString("\t<word tf=\"" + strconv.FormatFloat(float64(value), 'f', 5, 64) + "\">" + key + "</word>\n")
	}
	out.WriteString("</report>\n")

	fmt.Println(report.ID + " saved")

	return nil
}

func (report *GenericReport) getType(db *sql.DB, dbPrefix string) (reportType string) {
	reportType = "0"
	db.QueryRow("SELECT TYPE FROM bugs WHERE EXTERNAL_ID=?", dbPrefix+report.ID).Scan(&reportType)
	return reportType
}

func (report *GenericReport) DownloadFile() (err error) {

	//don't download file present on the disks
	if _, err := os.Stat(report.FilePath); os.IsNotExist(err) {

		// Create the file
		out, err := os.Create(report.FilePath)
		if err != nil {
			return err
		}
		defer out.Close()

		// Get the data
		resp, err := http.Get(report.Url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Writer the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
	}

	return nil
}
