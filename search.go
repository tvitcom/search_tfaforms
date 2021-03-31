package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DSN              = "searchforms:pass_to_searchforms@/searchforms"
	DRIVER           = "mysql"
	DBNAME           = "searchforms"
	FILES_ROOT_PATH  = "/srv/www/dl/prodapproot"
	FORMS_HTML_FPATH = "./html/forms.html"
)

var (
	empty string
	html  string
)

type (
	Forms struct {
		Id_forms int
		Name     string
		Test_uri string
	}
	Files struct {
		Id_forms      int
		Filepath      string
		Filepath_hash string
		Handle_by     string
	}
)

func main() {
	// Conn to database
	dbc, err := sql.Open(DRIVER, DSN)
	if err != nil {
		log.Fatal("DB is not connected")
	}
	if err = dbc.Ping(); err != nil {
		log.Fatal("DB is not responded")
	}
	defer dbc.Close()

	_ = gatherFormsHTMLInfo(FORMS_HTML_FPATH, dbc) // Forms info ---> db.forms

	_ = gatherFilesInfo(FILES_ROOT_PATH, ".tpl", "www.tfaforms.com/responses/processor", dbc) // Files info ---> db.files
}

// Parsing Forminfo form Html file
// Walk with goquery (like jquery)
// and insert into db forms tables
func gatherFormsHTMLInfo(fpath string, conn *sql.DB) error {
	bs, err := ioutil.ReadFile(FORMS_HTML_FPATH)
	if err != nil {
		log.Fatal("Not read file")
		panic(err)
	}
	html = string(bs)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		panic(err)
	}

	doc.Find("div.form-list-container").Find("div.form-info-container").Each(func(_ int, s *goquery.Selection) {
		str, _ := s.Attr("id")
		AssemblyId, _ := strconv.Atoi(str)
		AssemblyName := strings.TrimSpace(s.Find("h2.form-name").Text())
		url := "https://www.tfaforms.com/" + str
		form := Forms{Id_forms: AssemblyId, Name: AssemblyName, Test_uri: url}
		//fmt.Println("FORM is detected as:", AssemblyId)
		_ = insertFormInfo(conn, form)
	})
	return nil
}

// Searching and insert files info in database db.files
func gatherFilesInfo(rootFS, searchExtension, findText string, conn *sql.DB) error {
	filepath.Walk(rootFS, func(path string, info os.FileInfo, err error) error {
		fname := info.Name()
		if info.IsDir() || !strings.HasSuffix(fname, searchExtension) {
			return nil
		}
		if ok := isDetectedString(path, findText); ok {

			fmt.Println("-----------------------!!!---------------------->FILE is detected in:", path)

			fhash := GetMD5Hash(path)
			file := Files{ /*Id_files: 0,*/ Filepath: path, Filepath_hash: fhash, Handle_by: ""}
			_ = researchId(path, file, conn)
			_ = gatherHandler(path, ".php", conn)
		}
		return nil
	})
	return nil
}

func isDetectedString(fpath, needle string) bool {
	bs, err := ioutil.ReadFile(fpath)
	if err != nil {
		log.Fatal("Unreadable file " + fpath)
	}
	str := string(bs)
	if strings.Contains(str, needle) {
		return true
	}
	return false
}

func gatherHandler(fpath, searchExtension string, conn *sql.DB) error {
	_, findText := filepath.Split(fpath)
	filepath.Walk(FILES_ROOT_PATH, func(path string, info os.FileInfo, err error) error {
		fname := info.Name()
		if info.IsDir() || !strings.HasSuffix(fname, searchExtension) {
			return nil
		}
		if ok := isDetectedString(path, findText+"'"); ok {
			fmt.Println("HANDLER FOR:", fpath, "DETECTED for:", findText)
			fhash := GetMD5Hash(fpath)
			file := Files{ /*Id_files: 0,*/ Filepath: fpath, Filepath_hash: fhash, Handle_by: path}
			_ = insertFileInfo(conn, file)
		}
		return nil
	})
	return nil
}

func insertFormInfo(conn *sql.DB, f Forms) error {
	stmt, err := conn.Prepare("INSERT " + DBNAME + ".forms SET id_forms=?, name=?, test_uri=?")
	if err != nil {
		log.Fatal(err)
	}
	/*res*/ _, err = stmt.Exec(
		f.Id_forms,
		f.Name,
		f.Test_uri,
	)
	if err != nil {
		log.Fatal(err)
	}
	// affect, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println("INSERT FORM: Id_forms:", f.Id_forms, "- is inserted in DB:", affect)

	return nil
}

func insertFileInfo(conn *sql.DB, f Files) error {
	fmt.Println("TRY REPLACE FILEInfo:", f.Id_forms, f.Filepath, f.Filepath_hash, f.Handle_by)
	stmt, err := conn.Prepare("REPLACE " + DBNAME + ".files SET id_forms=?, filepath=?, filepath_hash=?, handle_by=?")
	if err != nil {
		log.Fatal(err)
	}
	res, err := stmt.Exec(
		f.Id_forms, f.Filepath, f.Filepath_hash, f.Handle_by,
	)
	if err != nil {
		log.Fatal(err)
	}
	affect, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("REPLACED in DB:", affect, "Id_forms:", f.Id_forms, f.Filepath)

	return nil
}

func AAAupdateFileId(conn *sql.DB, f Files) error {
	fmt.Println("TRY UPDATE ID FILEInfo:", f.Filepath, f.Filepath_hash, f.Id_forms)
	tx, err := conn.Begin()
	if err != nil {
		log.Fatal(err)
	}
	cmd := "UPDATE " + DBNAME + ".files SET id_forms=? WHERE filepath_hash=?"
	updateDate, err := tx.Prepare(cmd)
	if err != nil {
		log.Fatal(err)
	}
	res, err := updateDate.Exec(f.Id_forms, f.Filepath_hash)
	if err != nil {
		log.Fatal(err)
	}
	updateDate.Close()
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	affect, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("AFFECTED in DB:", affect, "UPDATED FILE:", f.Filepath)

	return nil
}

func updateHandler(conn *sql.DB, f Files) error {
	fmt.Println("TRY UPDATE HANDLER for:", f.Filepath, GetMD5Hash(f.Filepath))
	fhash := GetMD5Hash(f.Filepath_hash)
	stmt, err := conn.Prepare("REPLACE " + DBNAME + ".files SET id_forms=?, filepath=?, filepath_hash=?, handle_by=?")
	if err != nil {
		log.Fatal(err)
	}
	res, err := stmt.Exec(
		f.Id_forms, f.Filepath, fhash, f.Handle_by,
	)
	if err != nil {
		log.Fatal(err)
	}
	affect, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("REPLACED FILE:", f.Filepath, "- is inserted in DB:", affect)
	return err
}

func researchId(fpath string, file Files, conn *sql.DB) error {
	bs, err := ioutil.ReadFile(fpath)
	if err != nil {
		log.Fatal("Not read file")
		panic(err)
	}
	html = string(bs)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		panic(err)
	}
	doc.Find("form").Find("input#tfa_dbFormId").Each(func(_ int, s *goquery.Selection) {
		str, _ := s.Attr("value")
		AssemblyId, _ := strconv.Atoi(str)
		fmt.Println("RESEARCHED ID:", AssemblyId, "in", fpath)
		file.Id_forms = AssemblyId
		_ = insertFileInfo(conn, file)
	})
	return err
}

// Get md5 hash from string
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
