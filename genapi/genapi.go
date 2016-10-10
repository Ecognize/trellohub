package genapi

import (
  "net/http"
  "net/url"
  "encoding/json"
  "io/ioutil"
  "io"
  "bytes"
  "strings"
  "regexp"
  "log"
)

const REGEX_GH_OWNREPO string = "([a-z0-9][a-z0-9-]{0,38}[a-z0-9]/[a-z0-9][a-z0-9-]{0,38}[a-z0-9])?"
const REGEX_GH_REPO string = "^(?:https?://)?github.com/" + REGEX_GH_OWNREPO
const REGEX_GH_ISSUE string = REGEX_GH_REPO + "/issues/([0-9]*)"
// TODO: this ignores nesting, only top level is processed
// TODO: this might not work well with backslashes
const REGEX_GH_CHECK string = "(?:^|\\r\\n)- \\[([ x])\\] ([^\\r]*)"
// TODO: possibly separate GH and Trello version
const REGEX_GH_USER string = "(?i)@([a-z0-9][a-z0-9-]{0,38}[a-z0-9])"
const REGEX_GH_MAGIC string = "(?i)(?:close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved)[[:space:]]*" + REGEX_GH_OWNREPO + "#([0-9]*)"

type Set map[string]bool

/* Underlying type is reference */
func NewSet() Set {
  return make(Set)
}

func (set *Set) SetNameable(nm []string) {
  for k := range *set {
    delete(*set, k)
  }
  for _, v := range nm {
    (*set)[v] = true
  }
}

type CheckItem struct {
  Checked bool      `json:"-"`
  Text    string    `json:"name"`
  Id      string    `json:"id"`
  State   string    `json:"state"`
}

func (ch *CheckItem) FromTrello() {
  ch.Checked = ch.State == "complete"
}

func (ch *CheckItem) ForTrello() {
  if (ch.Checked) {
    ch.State = "complete"
  } else {
    ch.State = "incomplete"
  }
}

/* Reverse a dictionary (check if standar exist?) */
func DicRev(dic map[string]string) map[string]string {
  res := make(map[string]string)
  for k, v := range dic {
    res[v] = k
  }
  return res
}

/* Lua style strsub, replaces all matches of a regexp with what the callback returns
   Doesn't edit the original string */
// TODO handle bogus regexps
type StrSub_c func (v []string) string
func StrSub(source string, regtxt string, f StrSub_c) string {
  res := source
  re := regexp.MustCompile(regtxt)
  j := 0

  for {
    catch := re.FindStringSubmatchIndex(res[j:])
    if catch == nil {
      break
    }

  	// TODO proper slices maybe
  	lc := len(catch)/2
  	lo := catch[1] - catch[0]
    par := make([]string, lc)
    for i := 0; i < lc; i++ {
      par[i] = res[catch[i*2] + j:catch[i*2+1] + j]
    }
    rep := f(par)
    res = res[:catch[0] + j] + rep + res[catch[1] + j:]
    j = catch[1] + j - lo + len(rep)
  }
  return res
}

/* Replaces all occurences of @mentions between GitHub and Trello
   second parameter determines the dictionary */
func RepMentions(text string, dic map[string]string) string {
  return StrSub(text, REGEX_GH_USER, func (v []string) string {
    return "@"+dic[v[1]]
  })
}

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
  log.Printf("=> GET %s", rq)
  resp, err := http.Get(makeQuery(this, rq))
  processResponce(resp, err, &v)
}

/* Apparently no PUT or DELETE support in standard library, currently no output */
func genericRequest(this GenAPI, method string, rq string, rdr io.Reader) {
  log.Printf("=> %s %s", method, rq)
  client := &http.Client{}
  req, err := http.NewRequest(method, makeQuery(this, rq), rdr)
  /* TODO error handling */
  resp, err := client.Do(req)
  processResponce(resp, err, nil)
}

func GenPUT(this GenAPI, rq string) {
  genericRequest(this, "PUT", rq, nil)
}

func GenDEL(this GenAPI, rq string) {
  genericRequest(this, "DELETE", rq, nil)
}

/* Maybe generalise with other JSON func */
func GenDELJSON(this GenAPI, rq string, v interface{}) {
  // TODO JSON errors
  payload, _ := json.Marshal(&v)
  genericRequest(this, "DELETE", rq, bytes.NewReader(payload))
}

func GenPATCHJSON(this GenAPI, rq string, v interface{}) {
  // TODO JSON errors
  payload, _ := json.Marshal(&v)
  genericRequest(this, "PATCH", rq, bytes.NewReader(payload))
}

/* Pass a map, process structure later */
func GenPOSTForm(this GenAPI, rq string, v interface{}, f url.Values) { // TODO replace url.values with a struct
  log.Printf("=> POST %s", rq)
  resp, err := http.PostForm(makeQuery(this, rq), f)

  processResponce(resp, err, &v)
}

func GenPOSTJSON(this GenAPI, rq string, v interface{}, f interface{}) {
  log.Printf("=> POST %s", rq)
  /* TODO check json errors */
  payload, _ := json.Marshal(f)

  resp, err := http.Post(makeQuery(this, rq), "application/json", bytes.NewReader(payload))
  processResponce(resp, err, &v)
}

func processResponce(resp *http.Response, err error, v interface{}) {
  if err != nil {
    log.Fatal(err)
  } else {
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)

    if resp.StatusCode < 200 || resp.StatusCode > 299 {
      log.Printf("HTTP request returned response %d\n", resp.StatusCode)
      log.Fatalln(string(body[:]))
    } else if v != nil {
      /* TODO check json errors */
      json.Unmarshal(body, &v)
    }

    // log.Println(string(body[:]))
  }
}
