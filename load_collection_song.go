package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	// "github.com/k0kubun/pp"

	ccsv "github.com/lucmichalski/music-dataset/pkg/csv"
)

func main() {

	DB, err := gorm.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local", "root", "megamusic", "localhost", "3309", "dataset_music"))
	if err != nil {
		log.Fatal(err)
	}

	DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4")

	file, err := os.Open("/mnt/nasha/lucmichalski/music-dataset/feeds.itunes.apple.com/feeds/epf/v4/full/current/itunes20200501/itunes20200501/collection_song")
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(file)
	reader.Comma = '\x01'
	reader.LazyQuotes = false
	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	csvDataset, err := ccsv.NewCsvWriter("collection_song.csv", '\t')
	if err != nil {
		panic("Could not open `collection_song.csv` for writing")
	}

	counter := 0
	for {
		entry, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if perr, ok := err.(*csv.ParseError); ok && perr.Err == csv.ErrFieldCount {
				continue
			}
			continue
		}

		entry[5] = strings.Replace(entry[5], "\x02", "", -1)
		// pp.Println(entry)

		if len(entry) > 1 {
			csvDataset.Write(entry)
			csvDataset.Flush()
		}
		counter++
		if counter >= 15000 {
			// load data into the database
			loadData("collection_song.csv", DB)
			var err error
			csvDataset, err = ccsv.NewCsvWriter("collection_song.csv", '\t')
			if err != nil {
				panic("Could not open `collection_song.csv` for writing")
			}
			counter = 0
		}

	}

	csvDataset.Close()

}

func loadData(csvFile string, DB *gorm.DB) {
	fmt.Println("loading data from file...")
	mysql.RegisterLocalFile(csvFile)
	query := `LOAD DATA LOCAL INFILE '` + csvFile + `' IGNORE INTO TABLE collection_songs CHARACTER SET 'utf8mb4' FIELDS TERMINATED BY '\t' LINES TERMINATED BY '\n' (export_date,collection_id,song_id,track_number,volume_number,preorder_only);`
	fmt.Println(query)
	err := DB.Exec(query).Error
	if err != nil {
		log.Fatal(err)
	}
}
