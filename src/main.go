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
	"encoding/json"
	"errors"
//	u "github.com/YuseiRun/immich-backup-tool/src/utils"
	//	"strings"
)


type Config struct {
	ImmichUrl string `json:"immichUrl"`
	ImmichApiKey string `json:"immichApiKey"`
}


type Item struct {
	Id string `json:"id"`
}

type AssetResponseDto struct {
	
	Total int     `json:"total"`
	Count int     `json:"count"`
	Items []Item  `json:"items"`
	
}


type SearchAssetResponseDto struct {
    Assets AssetResponseDto `json:"assets"`
    //Total  int     `json:"total"`
    //Page   int     `json:"page"`
    //Size   int     `json:"size"`
}

func main (){



	//TODO: all the comments below
	//TODO: Separate functions into proper


	config, err := getConfigJson()

	if(err != nil){
		fmt.Sprintf("failed to get config in getImmichPhotos()")
	}

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

	//get newest entry in db date sync
	//lastSyncData, err := getLastSyncDate()
	
	//send last sync date here\
	//while lastSyncDate<currentDate loop the following
	getImmichPhotosAssetIds(config, time.Now())
	//downloadImmichAssets
	//


}

//func downloadImmichAssets(assets )

func cnxDb(db *sql.DB, sqlStr string, sqlTableName string){
	_, err := db.Exec(sqlStr)

	if( err != nil){
		log.Fatal(err)
	}
	log.Println( sqlTableName + " table connection success")

}

func jsonToType[T any](jsonByte[]byte, returnType *T) ( err error){
	err = json.Unmarshal(jsonByte, returnType)
	if err != nil{
		fmt.Sprintf("Error reading json")
		err = errors.New("Error reading json");
		return
	}
	return 
}

func getConfigJson() (config Config, err error) {
	configJson, err := os.ReadFile("config.json")

	if err != nil{
		fmt.Sprintf("Error finding config")
		err = errors.New("Error finding config");
		return
	}

	jsonToType(configJson, &config)
	return 

}



func getImmichPhotosAssetIds(config Config, syncDate time.Time){


	body := `
	{
		"updatedAfter":"`+ syncDate.AddDate(0,0,-20).Format("2006-01-02T15:04:05.000Z")+`", 

		"take":250
	}
	`
//		"updatedBefore":"`+ syncDate.AddDate(0,0,0).Format("2006-01-02T15:04:05.000Z") +`", 

	//https://api.immich.app/endpoints/search/searchAssets
	immichSearchMetaDataUrl := config.ImmichUrl + "/search/metadata";
//	fmt.Println(body + immichSearchMetaDataUrl)
	req, err := http.NewRequest("POST", immichSearchMetaDataUrl,bytes.NewBufferString(body))
	if err != nil {
		log.Println("failed to create request to immich server")
		os.Exit(1)
	}

	fmt.Println("Created Request")
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", config.ImmichApiKey)	

	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("Response Code:", resp.StatusCode)
		return
	}

	log.Println("sent request")
	
	
	
	var dto SearchAssetResponseDto
	err = json.NewDecoder(resp.Body).Decode(&dto)
	if err != nil {
		log.Fatal("Could not contact immich server")
	}
	
	fmt.Println(dto.Assets.Items[0].Id)//.Total)

}

func fileExists(filePath string){
	_, err := os.Stat(filePath);
	if os.IsNotExist(err) {
		os.Create(filePath)
	} else { 
		fmt.Sprintf("File path: '" + filePath + "'exists!")
	}
	
}
