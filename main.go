package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/qor/admin"
	"github.com/qor/assetfs"
	"github.com/qor/qor"
	"github.com/qor/qor/utils"
	"github.com/qor/validations"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	isHelp       bool
	isVerbose    bool
	isAdmin      bool
	isDataset    bool
	isDump       bool
	isLoadData   bool
	isTorProxy   bool
	parallelJobs int
	queueMaxSize = 100000000
	cachePath    = "./data/cache"
	DB           *gorm.DB
)

type Application struct {
	gorm.Model
	ExportDate        string
	ApplicationId     string
	Title             string
	RecommendedAge    string
	ArtistName        string
	SellerName        string
	CompanyUrl        string
	SupportUrl        string
	ViewUrl           string
	ArtworkUrlLarge   string
	ArtworkUrlSmall   string
	ItunesReleaseDate string
	Copyright         string
	Description       string
	Version           string
	ItunesVersion     string
	DownloadSize      string
}

type Collection struct {
	gorm.Model
	ExportDate          string
	CollectionId        int
	Name                string
	TitleVersion        string
	SearchTerms         string
	ParentalAdvisoryId  int
	ArtistDisplayName   string
	ViewUrl             string
	ArtworkUrl          string
	OriginalReleaseDate string
	ItunesReleaseDate   string
	LabelStudio         string
	ContentProviderName string
	Copyright           string
	PLine               string
	MediaTypeId         int
	IsCompilation       bool
	CollectionTypeId    string
}

type Song struct {
	gorm.Model
	ExportDate            string
	SongId                int
	Name                  string
	TitleVersion          string
	SearchTerms           string
	ParentalAdvisoryId    string
	ArtistDisplayName     string
	CollectionDisplayName string
	ViewUrl               string
	OriginalReleaseDate   string
	ItunesReleaseDate     string
	TrackLength           string
	Copyright             string
	PLine                 string
	PreviewUrl            string
	PreviewLength         string
}

func main() {
	pflag.IntVarP(&parallelJobs, "parallel-jobs", "j", 24, "parallel jobs.")
	pflag.BoolVarP(&isLoadData, "load", "l", false, "load data into file.")
	pflag.BoolVarP(&isTorProxy, "proxy", "x", false, "use tor proxy.")
	pflag.BoolVarP(&isDump, "dump", "p", false, "create csv dump.")
	pflag.BoolVarP(&isDataset, "dataset", "d", false, "generate dataset from db.")
	pflag.BoolVarP(&isAdmin, "admin", "a", false, "launch web admin.")
	pflag.BoolVarP(&isVerbose, "verbose", "v", false, "verbose mode.")
	pflag.BoolVarP(&isHelp, "help", "h", false, "help info.")
	pflag.Parse()
	if isHelp {
		pflag.PrintDefaults()
		os.Exit(1)
	}

	var err error
	if !isDump {
		DB, err = gorm.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local", "root", "megamusic", "localhost", "3309", "dataset_music"))
		if err != nil {
			log.Fatal(err)
		}

		DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4")
		DB.AutoMigrate(&Song{}, &Application{}, &Collection{})
		validations.RegisterCallbacks(DB)
		DB.LogMode(isVerbose)
	}

	// Initialize Admin
	if isAdmin {

		// Initialize AssetFS
		AssetFS := assetfs.AssetFS().NameSpace("admin")

		// Register custom paths to manually saved views
		AssetFS.RegisterPath(filepath.Join(utils.AppRoot, "./templates/qor/admin/views"))
		AssetFS.RegisterPath(filepath.Join(utils.AppRoot, "./templates/qor/media/views"))

		// Initialize Admin
		Admin := admin.New(&admin.AdminConfig{
			SiteName: "iTunes Dataset",
			DB:       DB,
			AssetFS:  AssetFS,
		})

		coll := Admin.AddResource(&Collection{}, &admin.Config{Menu: []string{"iTunes Management"}, Priority: -1})
		coll.IndexAttrs("ID", "ArtworkUrl", "Name", "Title", "Version", "Copyright", "LabelStudio")
		coll.Meta(&admin.Meta{Name: "ArtworkUrl", Valuer: func(record interface{}, context *qor.Context) interface{} {
			if c, ok := record.(*Collection); ok {
				result := bytes.NewBufferString("")
				tmpl, _ := template.New("").Parse(`<img src="{{.image}}">`)
				tmpl.Execute(result, map[string]string{"image": c.ArtworkUrl})
				return template.HTML(result.String())
			}
			return ""
		}})
		coll.UseTheme("grid")

		song := Admin.AddResource(&Song{}, &admin.Config{Menu: []string{"iTunes Management"}, Priority: -1})
		song.IndexAttrs("ID", "PreviewUrl", "Name", "ArtistDisplayName", "CollectionDisplayName", "OriginalReleaseDate")
		song.Meta(&admin.Meta{Name: "PreviewUrl", Valuer: func(record interface{}, context *qor.Context) interface{} {
			if s, ok := record.(*Song); ok {
				result := bytes.NewBufferString("")
				tmpl, _ := template.New("").Parse(`<audio controls="controls" src="{{.preview}}"></audio>`)
				tmpl.Execute(result, map[string]string{"preview": s.PreviewUrl})
				return template.HTML(result.String())
			}
			return ""
		}})

		app := Admin.AddResource(&Application{}, &admin.Config{Menu: []string{"iTunes Management"}, Priority: -1})
		app.IndexAttrs("ID", "ArtworkUrlSmall", "Title", "Version", "ArtistName", "SellerName", "ItunesReleaseDate")
		app.Meta(&admin.Meta{Name: "ArtworkUrlSmall", Valuer: func(record interface{}, context *qor.Context) interface{} {
			if a, ok := record.(*Application); ok {
				result := bytes.NewBufferString("")
				tmpl, _ := template.New("").Parse(`<img src="{{.image}}">`)
				tmpl.Execute(result, map[string]string{"image": a.ArtworkUrlSmall})
				return template.HTML(result.String())
			}
			return ""
		}})

		// initalize an HTTP request multiplexer
		mux := http.NewServeMux()

		// Mount admin interface to mux
		Admin.MountTo("/admin", mux)

		router := gin.Default()
		admin := router.Group("/admin", gin.BasicAuth(gin.Accounts{"music": "itunes"}))
		{
			admin.Any("/*resources", gin.WrapH(mux))
		}

		router.Static("/public", "./public")

		fmt.Println("Listening on: 9002")
		s := &http.Server{
			Addr:           ":9002",
			Handler:        router,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		s.ListenAndServe()
		os.Exit(1)

	}

	if isLoadData {
		loadData("/mnt/nasha/lucmichalski/music-dataset/feeds.itunes.apple.com/feeds/epf/v4/full/current/itunes20200501/itunes20200501/song", DB)
	}
}

func loadData(csvFile string, DB *gorm.DB) {
	fmt.Println("loading data from file...")
	mysql.RegisterLocalFile(csvFile)
	query := `LOAD DATA LOCAL INFILE '` + csvFile + `' INTO TABLE songs CHARACTER SET 'utf8mb4' FIELDS TERMINATED BY '^A' ENCLOSED BY '"' LINES TERMINATED BY '^B' IGNORE 35 LINES (export_date,song_id,name,title_version,search_terms,parental_advisory_id,artist_display_name,collection_display_name,view_url,original_release_date,itunes_release_date,track_length,copyright,p_line,preview_url,preview_length) SET created_at = NOW(), updated_at = NOW();`
	fmt.Println(query)
	err := DB.Exec(query).Error
	if err != nil {
		log.Fatal(err)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

/*
	Link        string   `gorm:"size:255;unique"`
	Alive       bool     `gorm:"index:alive"`
	StatusCode  int      `gorm:"index:status_code"`
	Name        string   `gorm:"index:name; type:longtext; CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	Path        string   `gorm:"index:path; type:longtext; CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	Title       string   `gorm:"type:longblob; CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci" sql:"type:longblob"`
	Description string   `gorm:"type:longblob; CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci" sql:"type:longblob"`
	Wap         string   `gorm:"type:longblob; CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci" sql:"type:longblob"`
*/
