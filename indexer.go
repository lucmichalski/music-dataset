package main

import (
	"fmt"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	manticoreHost = "127.0.0.1"
	manticorePort = 9313
)

func main() {

	DB, err := gorm.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local", "root", "megamusic", "localhost", "3309", "dataset_music"))
	if err != nil {
		log.Fatal(err)
	}

	DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4")

	// loadData("itunes.csv", DB)
	cl, _, err := initSphinx(manticoreHost, manticorePort)
	check(err)
	// services.Manticore = cl

	// index data
	cl, err := sql.Open("mysql", "@tcp(127.0.0.1:9306)/")
	if err != nil {
		panic(err)
	}

	var products []models.Product
	DB.Find(&products)
	for _, product := range products {
		// pp.Println(product)

		var deletedAt time.Time
		if product.Model.DeletedAt == nil {
			deletedAt = time.Date(2001, time.January, 01, 01, 0, 0, 0, time.UTC)
		} else {
			deletedAt = *product.Model.DeletedAt
		}

		query := fmt.Sprintf(`REPLACE into rt_itunes_song (id,created_at,updated_at,deleted_at,website,title,desc_seo,desc_fab,dcp,fab,ean13,iddcp,dim,url,price,star,carac,product_properties,description) VALUES ('%d','%d','%d','%d','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s')`,
			product.Model.ID,
			product.Model.CreatedAt.Unix(),
			product.Model.UpdatedAt.Unix(),
			deletedAt.Unix(),
			product.Website,
			putils.Escape(product.Title),
			putils.Escape(product.DescSeo),
			putils.Escape(product.DescFab),
			putils.Escape(product.DCP),
			putils.Escape(product.Fab),
			product.EAN13,
			product.IDDCP,
			putils.Escape(product.Dim),
			product.URL,
			product.Price,
			product.Star,
			putils.Escape(product.Carac),
			product.ProductProperties,
			putils.Escape(product.Description),
		)
		fmt.Println(query)
		res, err := cl.Exec(query)
		if err != nil {
			panic(err)
		}
		fmt.Println(res)
	}
	// os.Exit(1)

}

func initSphinx(host string, port uint16) (manticore.Client, bool, error) {
	cl := manticore.NewClient()
	cl.SetServer(host, port)
	status, err := cl.Open()
	if err != nil {
		return cl, status, err
	}
	return cl, status, nil
}
