package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"os/exec"
	"strings"
)

type album struct {
	id sql.NullString
}

type baseAlbum struct {
	id    sql.NullString
	title sql.NullString
}

type photo struct {
	id sql.NullString
}

type sizeVariant struct {
	shortPath sql.NullString
}

var lycheeUploadsRoot string
var exportDirectory string
var totalAlbums = 0
var totalPhotos = 0
var failedPhotos = 0

func main() {
	promptUserForFilePaths()

	db := promptForDatabase()

	rootAlbums := readRootAlbums(db)

	for _, a := range rootAlbums {
		export(a.id.String, exportDirectory, db)
	}

	fmt.Printf("Total Albums Exported: %d\n", totalAlbums)
	fmt.Printf("Total Photos Exported: %d\n", totalPhotos)
	fmt.Printf("Failed Photos: %d\n", failedPhotos)
}

func promptForDatabase() *sql.DB {
	var dbUrl string
	var dbPort string
	var dbName string
	var dbUsername string
	var dbPassword string
	var datasourceName string

	fmt.Println("Enter mysql db url/ip: ")
	_, err := fmt.Scanln(&dbUrl)
	if err != nil {
		log.Fatalf("Error getting input db url.  Error: %s", err.Error())
	}

	fmt.Println("Enter db port: ")
	_, err = fmt.Scanln(&dbPort)
	if err != nil {
		log.Fatalf("Error getting input db port.  Error: %s", err.Error())
	}

	fmt.Println("Enter database name: ")
	_, err = fmt.Scanln(&dbName)
	if err != nil {
		log.Fatalf("Error getting input db name.  Error: %s", err.Error())
	}

	fmt.Println("Enter database username: ")
	_, err = fmt.Scanln(&dbUsername)
	if err != nil {
		log.Fatalf("Error getting input db username.  Error: %s", err.Error())
	}

	fmt.Println("Enter database password: ")
	_, err = fmt.Scanln(&dbPassword)
	if err != nil {
		log.Fatalf("Error getting input db password.  Error: %s", err.Error())
	}

	datasourceName = dbUsername + ":" + dbPassword + "@tcp(" + dbUrl + ":" + dbPort + ")/" + dbName

	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		//WARNING!!!  LOGS PASSWORD IN PLAIN TEXT
		log.Fatalf("Error opening db connection.  Datasource: %s   Error: %s", datasourceName, err.Error())
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging db connection.  Datasource: %s   Error: %s", datasourceName, err.Error())
	} else {
		fmt.Println("Connection Successful")
	}

	return db
}

func promptUserForFilePaths() {
	fmt.Println("Enter Lychee uploads directory path:")
	_, err := fmt.Scanln(&lycheeUploadsRoot)
	if err != nil {
		log.Fatalf("error prompting user for lychee uploads directory path.  Error: %s\n", err.Error())
	}

	_, err = os.Stat(lycheeUploadsRoot)
	if err != nil {
		log.Fatalf("Error checking existance of lychee uploads directory: %s.  Error: %s\n",
			lycheeUploadsRoot, err.Error())
	}

	fmt.Println("Enter export root directory: ")
	_, err = fmt.Scanln(&exportDirectory)
	if err != nil {
		log.Fatalf("Error scanning for user input export directory.  %s\n", err.Error())
	}

	if _, err = os.Stat(exportDirectory); err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(exportDirectory, os.ModePerm)
			if err != nil {
				log.Fatalf("Error creating export root directory: %s.  %s\n", exportDirectory, err.Error())
			}
		} else {
			log.Fatalf("Error checking existance of export root directory: %s.  %s\n", exportDirectory, err.Error())
		}
	}
}

func export(albumId string, fsPath string, db *sql.DB) {
	totalAlbums++
	row := db.QueryRow("select id, title from base_albums where id = ?", albumId)

	var ba baseAlbum
	err := row.Scan(&ba.id, &ba.title)
	if err != nil {
		log.Fatalf("error querying for baseAlbum id: %s.  Error: %s\n", albumId, err.Error())
	}

	if !ba.title.Valid {
		log.Fatalf("Null title for album id: %s\n", albumId)
	}

	title := strings.Replace(ba.title.String, "/", "-", -1)
	title = strings.Replace(title, " ", "_", -1)

	fsPath = fsPath + "/" + title

	fmt.Printf("fsPath: %s\n", fsPath)

	err = os.Mkdir(fsPath, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating directory for album: %s\n", fsPath)
	}

	err = os.Chdir(fsPath)
	if err != nil {
		log.Fatalf("Error changing working directory %s\n", fsPath)
	}

	rows, err := db.Query("select id from photos where album_id = ?", albumId)
	if err != nil {
		log.Fatalf("Error querying for photos for albumId: %s.  Error: %s\n", albumId, err.Error())
	}

	photos := make([]photo, 0, 10)

	for rows.Next() {
		var p photo
		err = rows.Scan(&p.id)
		if err != nil {
			log.Fatalf("Error Scanning photo row for albumId: %s.  Error: %s\n", albumId, err.Error())
		}

		photos = append(photos, p)
	}

	for _, p := range photos {
		row = db.QueryRow("select short_path from size_variants where type = 0 AND photo_id = ?", p.id.String)

		var sv sizeVariant
		err = row.Scan(&sv.shortPath)
		if err != nil {
			log.Fatalf("Error querying size_variants for photoId: %s.  Error: %s\n", p.id.String, err.Error())
		}

		if !sv.shortPath.Valid {
			log.Fatalf("Short_path invalid for photo_id: %s\n", p.id.String)
		}

		photoPath := lycheeUploadsRoot + "/" + sv.shortPath.String

		fmt.Printf("Photo path: %s\n", photoPath)

		cp := exec.Command("rsync", "-avz", photoPath, fsPath)
		err = cp.Run()
		if err != nil {
			fmt.Printf("error copying photo from %s to %s.  Error: %s\n", photoPath, fsPath, err.Error())
			failedPhotos++
		} else {
			totalPhotos++
		}
	}

	rows, err = db.Query("select id from albums where parent_id = ?", albumId)
	if err != nil {
		log.Fatalf("Error querying for child albums albumId: %s.  Error: %s\n", albumId, err.Error())
	}

	childAlbums := make([]album, 0, 10)

	for rows.Next() {
		var a album
		err = rows.Scan(&a.id)
		if err != nil {
			log.Fatalf("Error scanning child albums.  AlbumId:  %s.  Error: %s\n", albumId, err.Error())
		}

		childAlbums = append(childAlbums, a)
	}

	for _, ca := range childAlbums {
		export(ca.id.String, fsPath, db)
	}
}

func readRootAlbums(db *sql.DB) []*album {
	rows, err := db.Query("select id from albums where parent_id is null")
	if err != nil {
		log.Fatalf("Error querying for root albums.  Error: %s\n", err.Error())
	}

	albums := make([]*album, 0, 10)

	for rows.Next() {
		var a album
		err = rows.Scan(&a.id)
		if err != nil {
			log.Fatalf("Error getting root albums: %s\n", err.Error())
		}

		albums = append(albums, &a)
	}

	return albums

}
