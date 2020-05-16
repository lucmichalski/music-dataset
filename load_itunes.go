package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	// "strconv"
	"strings"
	// "time"

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

	file, err := os.Open("/mnt/nasha/lucmichalski/music-dataset/feeds.itunes.apple.com/feeds/epf/v4/full/current/itunes20200501/itunes20200501/song")
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(file)
	reader.Comma = '\x01'
	reader.LazyQuotes = false
	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	csvDataset, err := ccsv.NewCsvWriter("itunes.csv", '\t')
	if err != nil {
		panic("Could not open `dataset.txt` for writing")
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

		entry[9] = strings.Replace(entry[9], " ", "-", -1)
		entry[10] = strings.Replace(entry[10], " ", "-", -1)
		entry[15] = strings.Replace(entry[15], "\x02", "", -1)

		if len(entry) > 1 {
			// pp.Println(entry)
			csvDataset.Write(entry)
			csvDataset.Flush()
		}
		counter++
		if counter >= 15000 {
			// load data into the database
			loadData("itunes.csv", DB)
			var err error
			csvDataset, err = ccsv.NewCsvWriter("itunes.csv", '\t')
			if err != nil {
				panic("Could not open `dataset.txt` for writing")
			}
			counter = 0
		}
	}

	csvDataset.Close()

}

func loadData(csvFile string, DB *gorm.DB) {
	fmt.Println("loading data from file...")
	// time.Sleep(10 * time.Second)
	mysql.RegisterLocalFile(csvFile)
	query := `LOAD DATA LOCAL INFILE '` + csvFile + `' IGNORE INTO TABLE songs CHARACTER SET 'utf8mb4' FIELDS TERMINATED BY '\t' LINES TERMINATED BY '\n' (export_date,song_id,name,title_version,search_terms,parental_advisory_id,artist_display_name,collection_display_name,view_url,original_release_date,itunes_release_date,track_length,copyright,p_line,preview_url,preview_length) SET created_at = NOW(), updated_at = NOW();`
	fmt.Println(query)
	err := DB.Exec(query).Error
	if err != nil {
		log.Fatal(err)
	}
}
