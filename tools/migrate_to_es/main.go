package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/olivere/elastic.v3"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

//meta type of Catalog
type Catalog struct {
	Link     string     `json:"link"`
	Name     string     `json:"name"`
	Children []*Catalog `json:"children"`
}

type StatElement struct {
	StringFeedbackCount string `json:"feedbackCount" bson:"feedbackCount"`
	StringPercentage    string `json:"percentage" bson:"percentage"`
	Count               int    `json:"c" bson:"-"`
	Percentage          int    `json:"p" bson:"-"`
}

type ShopInfo struct {
	Address   string    `json:"address,omitempty"`
	StartDate time.Time `json:"startDate,omitempty"`
	State     string    `json:"ur,omitempty"`
	Vendor    string    `json:"vendor,omitempty"`
}

//type of shop
type Shop struct {
	ID           int                     `json:"id" bson:"_id"`
	Name         string                  `json:"name" bson:"name"`
	Address      string                  `json:"address" bson:"address"`
	Dsc          string                  `json:"dsc" bson:"dsc"`
	IsActive     bool                    `json:"isActive" bson:"isActive"`
	IsNew        bool                    `json:"isNew" bson:"isNew"`
	IsExists     bool                    `json:"isExists" bson:"isExists"`
	Info         ShopInfo                `json:"info,omitempty" bson:"info,omitempty"`
	ShopStat     map[string]*StatElement `json:"stat" bson:"stat"`
	Catalogs     []*Catalog              `json:"catalogs" bson:"catalogs"`
	CatalogCount int                     `json:"catalogsCount"`
}

func changeCatalogName(c *Catalog) {
	c.Name = changeToLine.ReplaceAllString(c.Name, `_`)
	if len(c.Children) > 0 {
		for _, cc := range c.Children {
			changeCatalogName(cc)
		}
	}
}

var topicTops = []string{
	`Электроника`,
	`Компьютеры`,
	`Бытовая_техника`,
	`Дом_и_дача`,
	`Гардероб`,
	`Детские_товары`,
	`Красота_и_здоровье`,
	`Спорт_и_отдых`,
	`Авто`,
	`Подарки_и_цветы`,
	`Досуг_и_развлечения`,
	`Оборудование`,
	`Товары_для_офиса`,
}

var secondLevelTopic = map[string]map[string]bool{}

var getInt = regexp.MustCompile(`\d+`)
var changeToLine = regexp.MustCompile(`[ \(\)",\.-]`)

func changeStat(s *StatElement) {
	rawCount := getInt.FindString(s.StringFeedbackCount)
	if len(rawCount) != 0 {
		s.Count, _ = strconv.Atoi(rawCount)
	}
	rawProc := getInt.FindString(s.StringPercentage)
	if len(rawProc) != 0 {
		s.Percentage, _ = strconv.Atoi(rawProc)
	}
}

func main() {
	mgoSesssion, err := mgo.Dial("mongodb://localhost/shops")
	if err != nil {
		panic(err)
	}
	defer mgoSesssion.Close()

	client, err := elastic.NewClient()
	if err != nil {
		// Handle error
		panic(err)
	}
	info, code, err := client.Ping(elastic.DefaultURL).Do()
	if err != nil {
		// Handle error
		panic(err)
	}
	fmt.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)
	exists, err := client.IndexExists("yandex-market").Do()
	if err != nil {
		// Handle error
		panic(err)
	}
	if exists {
		_, err = client.DeleteIndex("yandex-market").Do()
		if err != nil {
			// Handle error
			panic(err)
		}
		_, err = client.CreateIndex("yandex-market").Do()
		if err != nil {
			// Handle error
			panic(err)
		}

	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex("yandex-market").Do()
		if err != nil {
			// Handle error
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	}

	var shop Shop
	iter := mgoSesssion.DB("").C("shops").Find(nil).Iter()
	for iter.Next(&shop) {
		shop.Name = strconv.Itoa(shop.ID)+"_"+changeToLine.ReplaceAllString(shop.Name, `_`)

		if len(shop.Catalogs) > 0 {
			for _, c := range shop.Catalogs {
				link, _ := url.Parse(c.Link)
				hid := link.Query().Get("hid")
				_, ok := secondLevelTopic[c.Name]
				if ok {
					secondLevelTopic[c.Name][link.Path+`?hid=`+hid] = true
				} else {
					secondLevelTopic[c.Name] = map[string]bool{link.Path + `?hid=` + hid: true}
				}
				changeCatalogName(c)
				c.Name = c.Name + "_" + hid + "_" + getInt.FindString(link.Path)
			}
			shop.CatalogCount = len(shop.Catalogs)
		}

		for _, s := range shop.ShopStat {
			changeStat(s)
		}
		post, err := client.Index().
			Index("yandex-market").
			Type("shop").
			BodyJson(shop).
			Id(strconv.Itoa(shop.ID)).
			Do()
		if err != nil {
			// Handle error
			panic(err)
		}
		fmt.Printf("Indexed shop %s to index %s, type %s\n", post.Id, post.Index, post.Type)
	}
	client.Flush().Do()
	client.Index().Index("yandex-market").Refresh(true).Do()
	if err := iter.Close(); err != nil {
		panic(err)
	}
	for cat, links := range secondLevelTopic {
		fmt.Printf("name=%s ", cat)
		for link, _ := range links {
			fmt.Printf("link=%s \n", link)
		}
	}
}
