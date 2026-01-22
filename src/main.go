package main

import (
 	_ "github.com/mattn/go-sqlite3"
	"fmt"
	"os"
	"os/exec"
	"log"
	"database/sql"
	"net/http"
	"bytes"
	"time"
	"encoding/json"
	"errors"
	"strings"
	"regexp"
	"io"
	"sync"
	"context"
 	"golang.org/x/sync/semaphore"
	"github.com/shirou/gopsutil/disk"
	//	"golang.org/x/sys/unix"
	"path/filepath"
  "runtime"

	//"github.com/gosuri/uilive"
)


type Config struct {
	ImmichUrl    	string  `json:"immichUrl"`
	ImmichApiKey 	string  `json:"immichApiKey"`
	DownloadLoc  	string  `json:"downloadLocation"`
	Concurrent    int     `json:"concurrentDownloads"`
	MaxDiskUsage	float64 `json:"maxDiskUsage"`
}


type Item struct {
	Id               string    `json:"id"`
	OriginalFileName string    `json:"originalFileName"`
	LocalDateTime    time.Time `json:"localDateTime"`
	UpdatedAt				 time.Time `json:"updatedAt"`
}

type SearchAssetResponseDto struct {
	Count     int     `json:"count"`
  NextPage  string  `json:"nextPage"`
	Total     int     `json:"total"`
	Items     []Item  `json:"items"`
	
}


type MetaDataResponseDto struct {
    Assets    SearchAssetResponseDto `json:"assets"`
    Total     int              `json:"total"`
}

var db *sql.DB
var config Config


func main (){
	//TODO: all the comments below
	//TODO: Separate functions into Processing
	
	var err error

	hasExif := exifInstalled()

	if !hasExif {
		fmt.Println("Please install exiftool on your system")
		return
	}


	config, err = getConfigJson()

	if(err != nil){
		log.Println("failed to get config in getImmichPhotos()")
		return
	}

	bo, err := verifyConfig()

	if(bo ==0 || err != nil){
		log.Println(err)
		return
	}

	if config.DownloadLoc == "THIS_LOCATION" {
		var input string
		path := getApplicationPath()
		path += "/immichPhotos"
		fmt.Printf("Using the current location will add all files in %s\nType [Y] to confirm, anything else will quit the program.",path)
		fmt.Scanln(&input)
		if input != "Y" {
			fmt.Println("Canceling program")
			return
		}
		fmt.Printf("Continuing....")
	}

	log.Println("started")
	dbPath :=  "../db";
	
	folderExists(dbPath)

	dbPath += "/database.db"
	fileExists(dbPath)
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
	    log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
	    log.Fatal(err)
	}
	

	defer db.Close()
	
	sqlLastSyncDB := `
	CREATE TABLE IF NOT EXISTS lastSync (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		lastSyncDtm Date,
		success String,
		totalSync INTEGER
	);
	`

	failedToDownloadTable :=` 
	CREATE TABLE IF NOT EXISTS failedAssets (
		assetId VARCHAR(127) NOT NULL,
		success INTEGER NOT NULL DEFAULT 0
	);
	`
//
//		id INTEGER NOT NULL PRIMARY KEY,
//	FOREIGN KEY (id) REFERENCES lastSync(id)
	
	
	cnxDb(db, sqlLastSyncDB, "lastSync")
	cnxDb(db, failedToDownloadTable, "failedAssets")

	//get newest entry in db date sync
	getDate := getLastSyncDate()
	//getDate=time.Now()
	//send last sync date here\
	//while lastSyncDate<currentDate loop the following
	pageNum :="1"
	
	folderExists(config.DownloadLoc)
	getImmichPhotosAssetIds(getDate,pageNum)
	lastSyncInsertSQL := "INSERT INTO lastSync(lastSyncDtm, success, totalSync) VALUES (?,?,?)"
	_, err = db.Exec(lastSyncInsertSQL, time.Now(), "SUCCESS", -1 )
		
	currentFailedCount := getCurrentFailedAssets()
	if( currentFailedCount != -1) {
		fmt.Printf("There are currently %d assets that need to be redownloaded\n", currentFailedCount)
	}

	//downloadImmichAssets(config,assetIds)
	log.Println("Downloads Complete!")	



}

func getApplicationPath() string {
	_, filename, _, _ := runtime.Caller(0)
  appDir := filepath.Dir(filename)

	lastSlash := strings.LastIndex(appDir, "/")
  if lastSlash == -1 {
		return appDir // No slash found, return original string
  }

  // Find the second-to-last occurrence of '/'
  secondToLastSlash := strings.LastIndex(appDir[:lastSlash], "/")
  if secondToLastSlash == -1 {
      return appDir // Only one slash, return original string
  }

  // Return the substring starting from the second-to-last slash + 1
	return appDir[:secondToLastSlash]

 }

func getCurrentFailedAssets() int {
	currentFailedCount:=-1 
	currentFailedToDownloadSQL:="SELECT COUNT (*) from failedAssets WHERE success = 0"
	
	err := db.QueryRow(currentFailedToDownloadSQL).Scan(&currentFailedCount)
	if err != nil {
		fmt.Println("There was a problem checking the failedAssets tabled.")
	}
	return currentFailedCount
}

func getLastSyncDate()(time.Time){
	var lsDate time.Time
	lastSyncDateSQL := "SELECT lastSyncDtm from lastSync WHERE success = 'SUCCESS' ORDER BY lastSyncDtm DESC LIMIT 1"
	err := db.QueryRow(lastSyncDateSQL).Scan(&lsDate)
	if err 	!= nil {
		log.Println(err)
		//if err == sql.ErrNoRows{
			log.Println("Setting first date as 1/1/1970")
			lsDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		//}	else {
		//	log.Panic("Error in retrieving rows")
		//}
	} else {
		fmt.Printf("Last Date Sync was: %s\n", lsDate.Format("2006-01-02 15:04:05"))
	}
	return lsDate
}

func downloadImmichAssets(assets []Item, totalCount int) {
	//need to get the download location
	log.Println(config.DownloadLoc)
	//create a new folder based on date provided
	folderExists(config.DownloadLoc)
	
	var wg sync.WaitGroup
	//guard:= make(chan struct {}, config.Concurrent)
	sem := semaphore.NewWeighted(int64(config.Concurrent))//make(chan struct, config.Concurrent)

	if(len(assets)<1){
		return 
	}
	for i := 0; i < len(assets); i++{
		folderExists(config.DownloadLoc + "/" +assets[i].LocalDateTime.Format("2006-01-02"))
		//guard <-struct{}{}
		wg.Add(1)
		
		go func(a Item){
			defer func(){ 
				//<-guard
				//defer wg.Done()
				ctx := context.Background()
				sem.Acquire(ctx,1)
				defer sem.Release(1)
		 		downloadAsset(a,i,totalCount, &wg)
			}()
		}(assets[i])
	 }

	//download all assets to the new folder
	wg.Wait()

	return 
}

func downloadAsset(asset Item, count int, total int, wg *sync.WaitGroup){ //sem chan struct{}, 
	defer wg.Done() //making sure we close the sync
		
	fmt.Printf("\rDownloading file number: %d/%d", count, total)
	filename := config.DownloadLoc+"/"+ asset.LocalDateTime.Format("2006-01-02")+"/"+asset.OriginalFileName;
	
	if(fileExists(filename)){
		//log.Println("\r\nFile exists at: " +filename +"\n")
		return
	}	
	//https://api.immich.app/endpoints/assets/downloadAsset
	immichSearchMetaDataUrl := config.ImmichUrl + "/assets/"+asset.Id+"/original";
	
	req, err := http.NewRequest("GET", immichSearchMetaDataUrl, nil)
	if err != nil {
		log.Println("failed to create request to immich server")
		os.Exit(1)
	}


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
		log.Println("Response Code:", resp.StatusCode)
		return
	}


	contentDisposition := resp.Header.Get("Content-Disposition")
	//filename := config.DownloadLoc+"/"+ asset.LocalDateTime.Format("2006-01-02")+"/fileName-"+time.Now().Format("2006-01-02-15:04:05")+".JPG";
	re := regexp.MustCompile(`^.*'`)
	if strings.Contains(contentDisposition, "filename*=") {
		parts:= strings.Split(contentDisposition, "filename*=")
		if len(parts) >1 {
			filename = config.DownloadLoc + "/" + asset.LocalDateTime.Format("2006-01-02") + "/" + re.ReplaceAllString(strings.Join(parts,""),"")
		}

	}
	
	//TODO: have a log/table of failed to download files. 
	//want users to know what didnt download, the ability to redownload (auto? leaning towards yes)
	//remove entries that were succesfully downloaded
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Could not create: " + filename)
		return
	}
	defer file.Close()

	_,err = io.Copy(file, resp.Body)

	if err != nil {
		fmt.Println("Could not copy to file: " + filename)
		errDelete := os.Remove(filename)
    if errDelete != nil {
			fmt.Println("Error deleting file: ", errDelete)
    }
		failedAssetsSQL := "INSERT into failedAssets(assetId) values (?)"
		_, err := db.Exec(failedAssetsSQL, asset.Id)
		if err != nil {
			log.Println("Could not record that %s was failed to download", asset.Id)
		}
	}	else {
		updateDate(filename, asset.LocalDateTime.Format("2006-01-02T15:04:05.000Z"))
	}
	//TODO: want to check the root location of the DownloadLoc
	//This so I am not checking the entirity of all of the drives connected
	//re=regexp.MustCompile('^/([^/]+)')
	usage, err := disk.Usage("/")// + re.FindStringSubmatch(config.DownloadLoc)[1])
	if(usage.UsedPercent > float64(config.MaxDiskUsage) ) {
		//TODO: investigate why error has !\n(MISSING)
		error := fmt.Sprintf("TOO MUCH SPACE USED\n Please clean up your disk or increase limit to be > %g%\n\nNow Exiting Program", usage.UsedPercent)
		log.Panic(error)
	}	
}

func exifInstalled() bool {
	cmd := exec.Command("exiftool", "-ver")
    output, err := cmd.Output()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return false
    }
		if strings.Contains(string(output),"command not food") {
			return false
		}
		return true
}


func updateDate(filename string, dateTaken string) error {//file *os.File, dateTaken string){
		cmd := exec.Command("exiftool", "-SubSecDateTimeOriginal=" +dateTaken, "-overwrite_original", filename)
		return cmd.Run()
}
 



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
	return config, err 

}

func verifyConfig() (int, error){
	errStr := "Your config.json is missing the following values: "
		
	if (config.ImmichUrl == ""){
		errStr += "immichUrl, "
	}   
	if( config.ImmichApiKey == ""){
		errStr += "immichApiKey, "
	} 
	if( config.MaxDiskUsage == 0.0){
		errStr += "maxDiskUsage, "
	}
	if( config.DownloadLoc == ""){
		errStr += "downloadLocation, "
	}
	if(config.Concurrent == 0){
		//if you're reading my code and wondering why I added the ", " at the end of the last string I have 2 reasons
		//1. I may add more configs in the future
		//2. I can just lop off the last 2 chars of this string without checking if this function was called
		errStr += "concurrentDownloads, "
	} 
	 
	if( len(errStr) > 55){
		return 0, errors.New(errStr[:len(errStr)-2]) //remove the last 2 chars
	}
	return 1, nil 
}


func getImmichPhotosAssetIds(syncDate time.Time, pageNum string)(l []Item){

	body := `
	{
		"updatedAfter":"`+ syncDate.Format("2006-01-02T15:04:05.000Z")+`" 
		,"page":`+ pageNum +`
		,"order":"asc"
	}
	`
//		"updatedBefore":"`+ syncDate.AddDate(0,0,0).Format("2006-01-02T15:04:05.000Z") +`", 

//		"take":250

	//https://api.immich.app/endpoints/search/searchAssets
	immichSearchMetaDataUrl := config.ImmichUrl + "/search/metadata";
//	fmt.Println(body + immichSearchMetaDataUrl)
	req, err := http.NewRequest("POST", immichSearchMetaDataUrl,bytes.NewBufferString(body))
	if err != nil {
		log.Println("failed to create request to immich server")
		os.Exit(1)
	}

	//log.Println("Created Request")
	
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
		log.Println("Response Code:", resp.StatusCode)
		return
	}

	var dto MetaDataResponseDto
	err = json.NewDecoder(resp.Body).Decode(&dto)
	if err != nil {
		log.Fatal("Could not contact immich server")
	}

	//fmt.Println("Total Count: ", dto.Assets.Count)
	//fmt.Println("Total Assets: ", dto.Assets.Total)
	//fmt.Println("Nextpage: ",  dto.Assets.NextPage)
	items := dto.Assets.Items

	if len(items) == 0 {
		return
	}
	lastSyncInsertSQL := "INSERT INTO lastSync(lastSyncDtm, success, totalSync) VALUES (?,?,?)"
	_, err = db.Exec(lastSyncInsertSQL, items[0].UpdatedAt, "STARTED", 0 )
	if( err != nil){
		log.Println("Failed to update lastSync table STARTED")
		return
	}
	downloadImmichAssets(items, dto.Assets.Total)
	lastSyncUpdateSQL := "UPDATE lastSync SET lastSyncDtm = ?, success = ? , totalSync = ? WHERE lastSyncDtm = ?"
	_, err = db.Exec(lastSyncUpdateSQL, items[len(items) -1].UpdatedAt, "SUCCESS", len(items), items[0].UpdatedAt)
	if( err != nil){
		log.Println("Failed to update lastSync table SUCCESS: ",err)
		return
	}
	if (dto.Assets.NextPage != "") {
		//items = append(items,
		getImmichPhotosAssetIds(syncDate, dto.Assets.NextPage)
		//...)
	} 
	return items
}

func fileExists(filePath string) bool{
	_, err := os.Stat(filePath);
	if os.IsNotExist(err) {

		file,err :=os.Create(filePath)
		if(err == nil){
			return false
		}

		defer file.Close()
		return false
	} else { 
		log.Println("\r\nFile path: '" + filePath + "'exists!\n")
		return true
	}
	
}
func folderExists(folderPath string){
	_, err := os.Stat(folderPath);
	if err != nil {
		os.MkdirAll(folderPath,0777)
	}
}
