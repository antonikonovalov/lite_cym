package main

import (
	"fmt"
	"strings"

	"bitbucket.org/poetofcode/antigate"
	"encoding/json"
	"errors"
	"flag"
	"github.com/fedesog/webdriver"
	"gopkg.in/mgo.v2"
	"log"
	"time"
)

const (
	ShopScript string = `
var captureUrl = "";
function captureDetection() {
    var captures = $('body > div.i-expander__content > div > div.content > div.form.form_state_image.form_error_no.form_audio_yes.i-bem.form_js_inited > form > img');
    if (captures.length > 0) {
        captureUrl = captures[0].src;
    }
};
captureDetection();

if (captureUrl.length > 0 ) {
    return {capture:captureUrl};
}

var shop = {};
var month = ['января','февраля','марта','апреля','мая','июня','июля','августа','сентября','октября','ноября','декабря'];
function getDateFromFore(dates) {
    console.log(dates);
    var mi = month.indexOf(dates[1]);
    if (mi !== -1) {
        mi = mi+1;
        raw = dates[2].slice(0,4)+"-"+mi+"-"+dates[0];
        console.log(raw);
        return new Date(Date.parse(raw)).toJSON();
    } else {
        return null;
    }

}

//can return error or shop
function setShopInfo(shop) {
    //checked is exists??
    shop.isExists = false;
    var rawNameShop = $('.headline > .title').text();
    if (rawNameShop === 'Магазин ') {
        return {error:"not found"};

    } else {
        shop.isExists = true;
        //удаляем 8 первых символов - 'Магазин '
        shop.name = rawNameShop.slice(8);
    }

    //checked is New??
    var newLabel = $('.headline__title-new');
    shop.isNew = false;
    if (newLabel.length === 1) {
        shop.isNew = true;
        return shop;
    }

    shop.dsc = $('.headline__footer').text();
    shop.isActive = true;
    // если магазин не размещается в этом ничего страшного нет
    // мы не можем получить инфу по категориям, но пожем собрать статы по отзывам
    // по этому идем дальше
    if (shop.dsc.includes('Магазин не размещается')) {
        shop.isActive = false;
    }

    shop.info = {};

    shop.info.vendor = $('.shop-info__header').text().slice(9);
    startDateEl = $('.shop-info__item');
    shop.info.startDate = getDateFromFore(startDateEl.text().slice(20).split(' '));
    shop.info.ur = startDateEl[0] &&  startDateEl[0].nextElementSibling ? startDateEl[0].nextElementSibling.textContent.trim() : '';
    shop.info.address = startDateEl[0] && startDateEl[0].nextElementSibling && startDateEl[0].nextElementSibling.nextElementSibling ? startDateEl[0].nextElementSibling.nextElementSibling.textContent.trim() : '';
    shop.stat = {};
    //todo
    countAllRateEl = $('.review-toolbar__count');
    $('.product-rating-stat > .rating-review').each(function (i,el){
        shop.stat[parseInt(el.childNodes[0].getAttribute('data-rate'))] = {
           percentage:el.childNodes[1].textContent,
           feedbackCount: el.childNodes[2].textContent
        };
    });

    return shop;
}

return setShopInfo(shop);
`
	CatalogScript string = `
var captureUrl = "";

function captureDetection() {
    var captures = $('body > div.i-expander__content > div > div.content > div.form.form_state_image.form_error_no.form_audio_yes.i-bem.form_js_inited > form > img');
    if (captures.length > 0) {
        captureUrl = captures[0].src;
    }
};


captureDetection();

if (captureUrl.length > 0 ) {
    return {capture:captureUrl};
}

function getMetaData (el) {
    return {
        link: el.href,
        name: el.text
    };
}

var catalogs = $('.b-menu.b-menu_layout_vert');

function processingOfChildren(el,parent) {
    if (el.childNodes.length === 2) {
    var listChildCatalog = Array.prototype.slice.call(el.childNodes[1].childNodes,0);
    if (listChildCatalog.length > 0) {
        parent.children = [];
        listChildCatalog.forEach(function (sel,i) {
            parent.children[i] = getMetaData(sel.querySelector('a.b-link'));
            if (sel.childNodes.length === 2) {
              processingOfChildren(sel,parent.children[i]);
            }
        });
    }
  }
}

function parseCatalogs() {
	var catalogData = [];
	catalogs.each(function (i,el) {
	  topNameEl = el.querySelector('a.b-link');
	  catalogData[i]=getMetaData(topNameEl);
	  processingOfChildren(el,catalogData[i]);
	});
	return catalogData;
}

return parseCatalogs();
`
)

type Capture struct {
	Capture string `json:"capture"`
}

//meta type of Catalog
type Catalog struct {
	Link     string     `json:"link"`
	Name     string     `json:"name"`
	Children []*Catalog `json:"children"`
}

type StatElement struct {
	FeedbackCount string `json:"feedbackCount" bson:"feedbackCount"`
	Percentage    string `json:"percentage" bson:"percentage"`
}
type ShopInfo struct {
	Address   string    `json:"address,omitempty"`
	StartDate time.Time `json:"startDate,omitempty"`
	State     string    `json:"ur,omitempty"`
	Vendor    string    `json:"vendor,omitempty"`
}

//type of shop
type Shop struct {
	ID       int                    `json:"id" bson:"_id"`
	Name     string                 `json:"name" bson:"name"`
	Address  string                 `json:"address" bson:"address"`
	Dsc      string                 `json:"dsc" bson:"dsc"`
	IsActive bool                   `json:"isActive" bson:"isActive"`
	IsNew    bool                   `json:"isNew" bson:"isNew"`
	IsExists bool                   `json:"isExists" bson:"isExists"`
	Info     ShopInfo               `json:"info,omitempty" bson:"info,omitempty"`
	ShopStat map[string]StatElement `json:"stat" bson:"stat"`
	Catalogs []*Catalog             `json:"catalogs" bson:"catalogs"`
}

func getVendorPage(vendorID int) string {
	return fmt.Sprintf(`https://market.yandex.ru/shop/%d/reviews`, vendorID)
}

func getCatalogPageByVandorID(vendorID int) string {
	return fmt.Sprintf(`https://market.yandex.ru/search?fesh=%d`, vendorID)
}

func processingOfCaptcha(url string, sess *webdriver.Session, cap *Capture) error {
	log.Print("START USING ANTIGATE FOR ",url)
	a := antigate.New(*antigateKey)
	captchaText, err := a.ProcessFromUrl(cap.Capture)
	if err != nil {
		return errors.New("can't process captcha from url by antigate url=" + cap.Capture + " err=" + err.Error())
	}
	inputEl, err := sess.FindElement(webdriver.CSS_Selector, `input.input__control.i-bem`)
	if err != nil {
		return errors.New("can't find input element for set capcha text: " + url + " error ::" + err.Error())
	}

	err = inputEl.SendKeys(captchaText)
	if err != nil {
		return errors.New("can't set captcha text by url " + url + " error ::" + err.Error())
	}
	err = inputEl.Submit()
	if err != nil {
		return errors.New("can't submit captcha answer:" + url + " error ::" + err.Error())
	}
	return nil
}

func SetVendorData(db *mgo.Session, sess *webdriver.Session, vendorID int) error {
	sess.Url(getVendorPage(vendorID))
	shopJson, err := sess.ExecuteScript(ShopScript, []interface{}{})
	if err != nil {
		return err
	}

	cap := &Capture{}
	err = json.Unmarshal(shopJson, cap)

	if err == nil && len(cap.Capture) > 0 {
		err = processingOfCaptcha(getVendorPage(vendorID), sess, cap)
		if err != nil {
			panic(err)
		}
		sess.Url(getVendorPage(vendorID))
		shopJson, err = sess.ExecuteScript(ShopScript, []interface{}{})
		if err != nil {
			return err
		}
	}

	shop := &Shop{}
	err = json.Unmarshal(shopJson, shop)
	if err != nil {
		return err
	}
	if !shop.IsExists {
		return errors.New("not found")
	}

	//при оставлении отзыва можно получить ссылку на магазин
	btnReviewEl, _ := sess.FindElement(webdriver.CSS_Selector, `.review-add-button`)
	btnReviewEl.Click()
	linkToShopEl, err := sess.FindElement(webdriver.CSS_Selector, ".headline__header > .title.title_size_32")
	if err != nil {
		log.Print("can't get link to shop for vendorID=", vendorID)
	} else {
		linkToShop, _ := linkToShopEl.Text()
		shop.Address = strings.TrimPrefix(linkToShop, `Мой отзыв о магазине `)
	}

	if shop.IsActive {
		newUrl := getCatalogPageByVandorID(vendorID)
		err = sess.Url(newUrl)
		if err != nil {
			return err
		}

		catalogsJson, err := sess.ExecuteScript(CatalogScript, []interface{}{})
		if err != nil {
			return err
		}
		cap2 := &Capture{}
		err = json.Unmarshal(catalogsJson, cap2)
		if err == nil && len(cap2.Capture) > 0 {
			err = processingOfCaptcha(getVendorPage(vendorID), sess, cap2)
			if err != nil {
				panic(err)
			}
			sess.Url(getCatalogPageByVandorID(vendorID))
			catalogsJson, err = sess.ExecuteScript(CatalogScript, []interface{}{})
			if err != nil {
				return err
			}
		}

		catalogs := []*Catalog{}
		err = json.Unmarshal(catalogsJson, &catalogs)
		if err != nil {
			return err
		}

		shop.Catalogs = catalogs
	}
	shop.ID = vendorID
	_, err = db.DB("").C("shops").UpsertId(shop.ID, shop)
	if err != nil {
		return err
	}
	return nil
}

var start = flag.Int("start", 3828, "set value from start id of shop")
var end = flag.Int("end", 3850, "set value to finish id of shop")
var pathToDriver = flag.String("pathToDriver", "/Users/antoniko/tensorflow/chromedriver", "set your path value")
var platform = flag.String("platform", "Mac", "set your platform")
var notCloseBrowser = flag.Bool("notCloseBrowser", false, "if your not want exist set true")
var antigateKey = flag.String("antigateKey", "", "set your antigate key")

func main() {
	flag.Parse()
	log.Print("start from", *start, " to ", *end)

	mgoSesssion, err := mgo.Dial("mongodb://localhost/shops")
	if err != nil {
		panic(err)
	}
	defer mgoSesssion.Close()

	chromeDriver := webdriver.NewChromeDriver(*pathToDriver)
	err = chromeDriver.Start()
	if err != nil {
		panic(err)
	}
	desired := webdriver.Capabilities{"Platform": *platform}
	required := webdriver.Capabilities{}
	session, err := chromeDriver.NewSession(desired, required)
	if err != nil {
		panic(err)
	}

	for vendorID := *start; vendorID < *end; vendorID++ {
		err := SetVendorData(mgoSesssion, session, vendorID)
		if err != nil {
			log.Print("error for processed ", vendorID, err.Error())
		}
		if (vendorID % 1000) == 0 {
			log.Print("processed ", vendorID)
		}
	}

	if !*notCloseBrowser {
		session.Delete()
		chromeDriver.Stop()
	}
	log.Print("processed ", *end-*start)
}
