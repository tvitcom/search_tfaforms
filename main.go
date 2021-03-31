package main

import (
	// "os"
	"fmt"
	"log"
	"net/http"
	"strconv"
    "crypto/tls"
	"database/sql"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"my.localhost/funny/dct_searchforms/models"
)

const (
	PRODMODE         = true
	DIRSEP           = "/"
	DSN              = "searchforms:pass_to_searchforms@/searchforms"
	DRIVER           = "mysql"
	DBNAME           = "searchforms"
	DSNS             = "dlprod:pass_to_dlprod@/information_schema"
	DSNT             = "dlprod:pass_to_dlprod@/dlprod"
	DBNAMET          = "dlprod"
	TPROTO           = "http://"
	FILES_ROOT_PATH  = "/srv/www/dl/prodapproot"
	FORMS_HTML_FPATH = "./html/forms.html"
)

var (
	empty string
	html  string
	err   error
	db,dbs,dbt *sql.DB
)

type Env struct {
    db,dbs,dbt *sql.DB
}

func init() {
	models.PRODMODE = PRODMODE

}

func main() {
	// Init the connections pool to database
	db, err := models.InitDB(DRIVER, DSN)
	if err != nil {
        log.Panic(err)
    }
	dbs, err = models.InitDB(DRIVER, DSNS) //Open cooonect pool for informational_schema getting
	if err != nil {
        log.Panic(err)
    }
	dbt, err = models.InitDB(DRIVER, DSNT)
	if err != nil {
        log.Panic(err)
    }
    env := &Env{db: db, dbs: dbs, dbt: dbt}
	// Run web server for observ current progress results:
	fmt.Println("Web server start on 127.0.0.1:3000")
	http.HandleFunc("/", env.homeHandler)
	http.ListenAndServe(":3000", nil)
fmt.Println("Scan start!")
	// !!!ONLY FOR MONITORING!!!
	// Observ current available links:
	// columnsPool, _ := models.ShowResearchedLinks(db)
	// if err != nil {
	// 	log.Panic(err)
	// }
	// fmt.Println("RESEARCHED COLUMNS:")
	// for _, v := range columnsPool {
	// 	fmt.Println(v.Id_forms, v.Live_uri)
	// }

	//Gather all tables, columns with varchar type:
	allVarColumns, err := models.GatherAllVarcharTablesColumns(env.dbs, DBNAMET)
	if err != nil {
		log.Panic(err)
	}
	if !PRODMODE {
		fmt.Println("FULL LIST URLS:")
		for _, u := range allVarColumns {
			fmt.Println(u.Table, u.Column)
		}
	}

	// For each tables.column research about applicable it. Check if it may be link: Not spaces and dashes
	//!!!TEST ONLY:
	// allVarColumns := []models.VarcharColumn{{Table:"ps_cms_lang", Column:"link_rewrite"}}
	pathes := make([]string, 0)
	for _, v := range allVarColumns {
		links, _ := models.ExtractValidPathes(env.dbt, v.Table, v.Column)
		if len(links) > 0 {
			pathes = append(pathes, links...)
		}
	}
	if !PRODMODE {
		fmt.Println("FULL LIST URLS:")
		for _, l := range pathes {
			fmt.Println(l)
		}
	}

	// Make variates of path for uri scannning
	urls := additionalPathPrefixes(pathes)
	TSITES := []string{
		"dldev.wwwls4.a2hosted.com",
		"sf.dev.wwwls4.a2hosted.com",
		// "bathauthority.dev.wwwls4.a2hosted.com",
		// "lowes.dev.wwwls4.a2hosted.com",
		// "lowespro.dev.wwwls4.a2hosted.com",
		// "ferguson.dev.wwwls4.a2hosted.com",
		"customglass.dev.wwwls4.a2hosted.com",
		// "homedepot.dev.wwwls4.a2hosted.com",
		// "homedepotca.dev.wwwls4.a2hosted.com",
		"outlet.dev.wwwls4.a2hosted.com",
	}
	urls = additionalUrlPrefixes(urls, TSITES)
	if !PRODMODE {
		fmt.Println("FULL LIST URLS:")
		for _, u := range urls {
			fmt.Println(u)
		}
	}
	for _, fullurl := range urls {
		// Scan and And save into database forms
		//!!!TEST ONLY:
		// form.Id_forms = 4622318
		// form.Live_uri = "https://dldev.a2hosted.com/content/technical-documentation-and-manuals"
		// r, _ := models.InsertDetectedForms(db, form)
		// fmt.Println("END TEST: UPDATED:", r)
		//err = tryScanByLink(TPROTO+TSITES[i]+"/become-a-dealer", env.db)
		err = tryScanByLink(TPROTO+fullurl, db)
			if err != nil {
			log.Panic(err)
		}
	}
}

func (env *Env) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	// Observ all collumn in database:
	columnsPool , _ := models.ShowResearchedLinks(env.db)
	if err != nil {
		http.Error(w, http.StatusText(507), 507)
		return
	}
	fmt.Fprintln(w, "RESEARCHED COLUMNS:")
	for _, v := range columnsPool {
		fmt.Fprintln(w, v.Id_forms,v.Live_uri)
	}
}

func additionalUrlPrefixes(links []string, sites []string) []string {
	united := make([]string, 0)
	if !PRODMODE {
		fmt.Println("PREFIXES:", sites)
	}
	for _, l := range links {
		for _, site := range sites {
			u := site
			if l != "/" {
				u = u + l
			}
			united = append(united, u)
		}
	}
	return united
}

// Transformation url - path from: some-link to: /some-link
func additionalPathPrefixes(links []string) []string {
	united := make([]string,0)
	prefix := make([]string, 2)
	prefix[0] = "/"
	prefix[1] = "/content/"
	if !PRODMODE {
		fmt.Println("PREFIXES:", prefix)
	}
	for _, v := range links {
		for _, p := range prefix {
			u := p + v
	        if !PRODMODE {
			    fmt.Println(u)
            }
			united = append(united, u)
		}
	}
	return united
}

func tryScanByLink(link string, conn *sql.DB) error {
	// Request the HTML page.
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{Transport: tr}
	res, err := client.Get(link)
	if err != nil {
		log.Panic(err)
	}
	
	log.Println(link, res.StatusCode)
	
	defer res.Body.Close()
	if res.StatusCode == 200 {
        // Load the HTML document
        doc, err := goquery.NewDocumentFromReader(res.Body)
        if err != nil {
            log.Panic(err)
        }
        doc.Find("form").Find("input#tfa_dbFormId").Each(func(_ int, s *goquery.Selection) {
            str, _ := s.Attr("value")
            AssemblyId, _ := strconv.Atoi(str)
            fmt.Println("RESEARCHED ID:", AssemblyId, "in", link)
            var f models.Forms
            f.Id_forms = AssemblyId
            f.Live_uri = link
            fmt.Println("DETECTED on", link, "ID:", AssemblyId)
            _, err = models.InsertDetectedForms(conn, f)
            if err != nil {
                log.Panic(err)
            }
        })
	}
	return nil
}
