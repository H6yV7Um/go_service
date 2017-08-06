package es

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

//type EsColumnDoc
type EsColumnField map[string]interface{}
type EsColumnDoc struct {
	Index   string  `json:"_index"`
	Types   string  `json:"_type"`
	Id      string  `json:"_id"`
	Version int     `json:"_version"`
	Found   bool    `json:"found"`
	Score   float64 `json:"_score"`
	/*Fields  struct {
		Oid  []string `json:"oid"`
		Tags []string `json:"tags"`
	} `json:"fields"`*/
	Source map[string]interface{} `json:"_source"`
}

func ColumnGet(oid string, fields []string) (EsColumnDoc, error) {
	var esData EsColumnDoc
	url := ES_URL + oid
	switch checkFields(fields) {
	case -1:
		return esData, errors.New("someone field is unvalidated")
	case 1:
		url = fmt.Sprintf("%s?_source_include=%s", url, strings.Join(fields, ","))

	}
	//return esData, errors.New((url))

	response, err := http.Get(url)
	if err != nil {
		return esData, errors.New("http err")
	}
	//defer log.Println(err)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return esData, errors.New("http server error:" + strconv.Itoa(response.StatusCode))
	}
	body, _ := ioutil.ReadAll(response.Body)

	//return strJson, nil
	if err := json.Unmarshal(body, &esData); err != nil {
		return esData, err
	}
	return esData, nil

	//return string(body), nil
}
