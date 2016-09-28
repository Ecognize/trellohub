package main

import (
  "net/http"
  "net/url"
  "encoding/json"
  "io/ioutil"
  "bytes"
  "strings"
  // "fmt"
)

/* Generalised functions like JSON decoding or lower level http work */
type GenAPI interface {
  AuthQuery() string  // Authentication query, keys, tokens etc
  BaseURL()   string  // URL base for REST
}

func makeQuery(this GenAPI, rq string) string {
  var delim string
  if strings.Contains(rq, "?") {
    delim = "&"
  } else {
    delim = "?"
  }
  return this.BaseURL() + rq + delim + this.AuthQuery()
}

/* HTTP method funcs basically all do the same, they compose the query and
   try to extract JSON output */
func GenGET(this GenAPI, rq string, v interface{}) {
  resp, err := http.Get(makeQuery(this, rq))
  processResponce(resp, err, &v)
}

/* Apparently no PUT support in standard library, currently no output */
func GenPUT(this GenAPI, rq string) {
  client := &http.Client{}
  req, err := http.NewRequest("PUT", makeQuery(this, rq), nil)
  /* TODO error handling */
  resp, err := client.Do(req)
}

/* Pass a map, process structure later */
func GenPOSTForm(this GenAPI, rq string, v interface{}, f url.Values) { // TODO replace url.values with a struct
  resp, err := http.PostForm(makeQuery(this, rq), f)

  processResponce(resp, err, &v)
}

func GenPOSTJSON(this GenAPI, rq string, v interface{}, f interface{}) {
  /* TODO check json errors */
  payload, _ := json.Marshal(f)

  resp, err := http.Post(makeQuery(this, rq), "application/json", bytes.NewReader(payload))
  processResponce(resp, err, &v)
}

func processResponce(resp *http.Response, err error, v interface{}) {
  // TODO check if resp is 200 and err is ok
  defer resp.Body.Close()
  body, _ := ioutil.ReadAll(resp.Body)

  /* TODO check json errors */
  json.Unmarshal(body, &v)

  // fmt.Println(string(body[:]))
}
