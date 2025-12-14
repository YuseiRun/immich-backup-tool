package main

import (
 	_ "github.com/mattn/go-sqlite3"
	"fmt"
	"os"
	"log"
	"database/sql"
	"net/http"
	"bytes"
	"time"
	//	"strings"
)


type Config struct{
	immichUrl string `json:"immichUrl"`,
	immichApiKey string `json:"immichApiKey`
}


func main (){


	fmt.Sprintf("started")
	const dbPath =  "../db/database.db";
	fileExists(dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil{
		log.Fatal(err)
	}

	defer db.Close()
	
	sqlLastSyncDB := `
	CREATE TABLE IF NOT EXISTS lastSync (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		lastSyncDtm Date,
		success String,
		totalSync int 
	);
	`

	sqlImageNameDB :=` 
	CREATE TABLE IF NOT EXISTS images (
		id INTEGER NOT NULL PRIMARY KEY,
		imageName VARCHAR(127),
		FOREIGN KEY (id) REFERENCES lastSync(id)
	);
	`
	cnxDb(db, sqlLastSyncDB, "lastSync")
	cnxDb(db, sqlImageNameDB, "image")

	//_, err = db.Exec(sqlImageNameDB)

	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Println("image table connection success")

	getImmichPhotos(time.Now())

}
func cnxDb(db *sql.DB, sqlStr string, sqlTableName string){
	_, err := db.Exec(sqlStr)

	if( err != nil){
		log.Fatal(err)
	}
	log.Println( sqlTableName + " table connection success")

}

func getImmichPhotos(syncDate time.Time){
	
	config, err := os.ReadFile("config.json")

	if err != nill{
		fmt.Sprintf("Error reading config")
		return
	}


	fmt.Println(syncDate.Unix())
	//"updatedAfter" :  LAST_SYNC_DATE
	// "take" : X
	body := `{"updatedAfter": "", "take": 250}`
	//https://api.immich.app/endpoints/search/searchAssets
	req, err := http.NewRequest("POST",config.immichUrl + "/search/metadata",bytes.NewBufferString(body))
	if err != nil {
		fmt.Sprintf("failed to contact to immich server")
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key","KEY")	

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()





}

func fileExists(filePath string){
	_, err := os.Stat(filePath);
	if os.IsNotExist(err) {
		os.Create(filePath)
	} else { 
		fmt.Sprintf("File path: '" + filePath + "'exists!")
	}
	
}
