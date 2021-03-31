package main

import (
    "log"
	"github.com/PuerkitoBio/goquery"
	"strings"
	"io/ioutil"
)

var  (
	html string
	iHtml int
)

func main() {
	//Read parsing string form Html file
	bs, err := ioutil.ReadFile("./html/forms.html")
    if err != nil {
        log.Fatal("Not read file")
        panic(err)
    }
    html = string(bs)
    
    //Walk with goquery (like jquery)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		panic(err)
	}

	doc.Find("div.form-list-container div.form-info-container").Each(func(iHtml int, s *goquery.Selection) {
		AssemblyId, _ := s.Attr("id")
		AssemblyName := s.Find("h2.form-name").Text()
		result := strings.Replace(AssemblyName, " ", "", -1)
		print(result,",",AssemblyId,",https://www.tfaforms.com/"+AssemblyId)
	})
}
