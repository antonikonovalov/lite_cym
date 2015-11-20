package main

import (
	"encoding/json"
	"testing"
)

var shop = []byte(`{"dsc":"Рейтинг 4 из 5 составлен на основе 2350 оценок покупателей и данных службы качества МаркетаОбщий рейтинг на основе 2350 оценок покупателей и данных службы качества Маркета","info":{"address":"Юридический адрес: 115280, г. Москва, ул. Ленинская Слобода, д. 19","startDate":{},"ur":"Связной Логистика, ОГРН 1057748731336","vendor":"                Продавец Связной Логистика"},"isActive":true,"isExists":true,"isNew":false,"name":"СВЯЗНОЙ","stat":{"1":{"feedbackCount":"5288 отзывов","precentage":"14%"},"2":{"feedbackCount":"1395 отзывов","precentage":"4%"},"3":{"feedbackCount":"1565 отзывов","precentage":"4%"},"4":{"feedbackCount":"4793 отзыва","precentage":"13%"},"5":{"feedbackCount":"24644 отзыва","precentage":"65%"}}}`)

func TestUnmarshelShop(t *testing.T) {
	shopObj := &Shop{}
	err := json.Unmarshal(shop,shopObj)
	if err != nil {
		t.Error(err)
	}
}
