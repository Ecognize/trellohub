package main

import (
  "net/http"
  "encoding/json"
  "io/ioutil"
)

/* Generalised functions like JSON decoding or lower level http work */
type GenAPI interface {
  AuthQuery() string  // Authentication query, keys, tokens etc
  BaseURL()   string  // URL base for REST
}

/* HTTP method funcs basically all do the same, they compose the query and
   try to extract JSON output */
func GenGET(this GenAPI, rq string, v interface{}) {
  resp, err := http.Get(this.BaseURL() + rq + this.AuthQuery())
  processResponce(resp, err, &v)
}

func processResponce(resp *http.Response, err error, v interface{}) {
  // TODO check if resp is 200 and err is ok
  defer resp.Body.Close()
  body, _ := ioutil.ReadAll(resp.Body)
  json.Unmarshal(body, &v)
}
