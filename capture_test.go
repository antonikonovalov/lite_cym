package main
import (
	"testing"
	"bitbucket.org/poetofcode/antigate"
	"fmt"
)

var captureUrl = `http://market.yandex.ru/captchaimg?aHR0cDovL25hLmNhcHRjaGEueWFuZGV4Lm5ldC9pbWFnZT9rZXk9YzJFcFYxMzBBSXQ2YVFpODJiVDdnckgxbWR5Zk1SS2I,_0/1448120301/c1b76a35a94d62b445df02f820342ba8_707f293360669b754fb1855ff1c7ec15`

func TestEquals (t *testing.T) {
	fromBrowser := "http://market.yandex.ru/captchaimg?aHR0cDovL25hLmNhcHRjaGEueWFuZGV4Lm5ldC9pbWFnZT9rZXk9YzF0NFdidFRrTENHY2RXWTl6Y2xrcE1DWkhvVks0Q1k,_0/1448129839/cb6d8d0c011a3fccd65c45fa1c879db6_4070c6c3bd691ed4ffef5a4a3b2d2ce0"
	fromScipt := "http://market.yandex.ru/captchaimg?aHR0cDovL25hLmNhcHRjaGEueWFuZGV4Lm5ldC9pbWFnZT9rZXk9YzF0NFdidFRrTENHY2RXWTl6Y2xrcE1DWkhvVks0Q1k,_0/1448129839/cb6d8d0c011a3fccd65c45fa1c879db6_4070c6c3bd691ed4ffef5a4a3b2d2ce0"
	if fromBrowser != fromScipt {
		t.Error("Expected equals - but it isn't!")
	}
}

func TestCapture (t *testing.T) {
	a := antigate.New("")

	// From URL
	captcha_text, _ := a.ProcessFromUrl(captureUrl)
	fmt.Println("from url:", captcha_text)
}
