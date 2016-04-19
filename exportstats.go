package exportstats

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/simonz05/util/handler"
	"github.com/simonz05/util/log"
)

type TimeUnit string

const (
	Minute TimeUnit = "m"
	Hour            = "h"
	Day             = "d"
	Week            = "w"
	Month           = "M"
	Year            = "y"
)

func ParseTimeUnit(v string) (TimeUnit, error) {
	switch v {
	case "m", "minute", "minutes":
		return Minute, nil
	case "h", "hour", "hours":
		return Hour, nil
	case "d", "day", "days":
		return Day, nil
	case "w", "week", "weeks":
		return Week, nil
	case "M", "month", "months":
		return Month, nil
	case "y", "year", "years":
		return Year, nil
	default:
		return "", errors.New("Unkown TimeUnit " + v)
	}
}

type Timeframe struct {
	DurationValue int
	DurationUnit  TimeUnit
	IntervalValue int
	IntervalUnit  TimeUnit
	Start         *time.Time
}

func (tf *Timeframe) UnmarshalJSON(data []byte) error {
	// trim " in beginning and end of JSON string
	if len(data) >= 2 {
		data = data[1 : len(data)-1]
	}

	_tf, err := ParseTimeframe(string(data))

	if err != nil {
		return err
	}

	*tf = _tf
	return nil
}

func (tf Timeframe) String() string {
	if tf.Start == nil {
		return fmt.Sprintf("%d%s@%d%s", tf.DurationValue, tf.DurationUnit, tf.IntervalValue, tf.IntervalUnit)
	}
	return fmt.Sprintf("%d%s@%d%s-%d", tf.DurationValue, tf.DurationUnit, tf.IntervalValue, tf.IntervalUnit, tf.Start.Unix())
}

func (tf Timeframe) Format() string {
	return fmt.Sprintf("%d%s%d%s", tf.DurationValue, tf.DurationUnit, tf.IntervalValue, tf.IntervalUnit)
}

var parseTimeframeRe = regexp.MustCompile("([0-9]+)([a-z]+)([0-9]+)([a-z]+)")

func ParseTimeframe(v string) (Timeframe, error) {
	tf := Timeframe{}
	parts := strings.Split(v, " ")

	if len(parts) != 5 {
		parts = make([]string, 5)
		match := parseTimeframeRe.FindAllStringSubmatch(v, -1)
		if match == nil || len(match) != 1 || len(match[0]) != 5 {
			return tf, errors.New("Parse Timeframe error")
		}
		parts[0] = match[0][1]
		parts[1] = match[0][2]
		parts[3] = match[0][3]
		parts[4] = match[0][4]
	}

	var err error
	tf.DurationValue, err = strconv.Atoi(parts[0])

	if err != nil {
		return tf, err
	}

	tf.DurationUnit, err = ParseTimeUnit(parts[1])

	if err != nil {
		return tf, err
	}

	tf.IntervalValue, err = strconv.Atoi(parts[3])

	if err != nil {
		return tf, err
	}

	tf.IntervalUnit, err = ParseTimeUnit(parts[4])

	if err != nil {
		return tf, err
	}

	return tf, nil
}

func MustParseTimeframe(v string) Timeframe {
	tf, err := ParseTimeframe(v)
	if err != nil {
		panic(err.Error())
	}
	return tf
}

type Point struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

func (p *Point) String() string {
	return fmt.Sprintf("(v: %v, t: %v)", p.Value, p.Time)
}

func (p *Point) ToCSV() []string {
	v := strconv.FormatFloat(p.Value, 'f', 6, 64)
	t := strconv.FormatInt(p.Time, 10)
	return []string{v, t}
}

type Dataset struct {
	Name      string    `json:"name"`
	Timeframe Timeframe `json:"timeframe"`
	Points    []*Point  `json:"points"`
}

func (ds *Dataset) String() string {
	return fmt.Sprintf("%s: %s - %v", ds.Name, ds.Timeframe, ds.Points)
}

type Stat struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Public  bool   `json:"public"`
	Counter bool   `json:"counter"`
}

var NotFoundErr error = errors.New("Stat not found")

type StatFetcher interface {
	Get(name string, tf Timeframe) (*Dataset, error)
}

type StatHatFetcher struct {
	AccessToken string
	baseURI     string
}

func NewStatHatFetcher(accessToken string) *StatHatFetcher {
	return &StatHatFetcher{
		AccessToken: accessToken,
		baseURI:     "https://www.stathat.com/x/" + accessToken,
	}
}

func (sh *StatHatFetcher) getStat(name string) (*Stat, error) {
	uri := sh.baseURI + "/stat?name=" + name
	log.Println(uri)
	res, err := http.Get(uri)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, NotFoundErr
	}

	dec := json.NewDecoder(res.Body)
	target := new(Stat)
	err = dec.Decode(target)

	if err != nil {
		return nil, err
	}

	return target, nil
}

func (sh *StatHatFetcher) Get(name string, tf Timeframe) (*Dataset, error) {
	stat, err := sh.getStat(name)
	if err != nil {
		return nil, err
	}

	uri := sh.baseURI + "/data/" + stat.ID + "?t=" + tf.Format()

	if tf.Start != nil {
		uri += "&start=" + strconv.Itoa(int(tf.Start.Unix()/1000))
	}

	log.Println(uri)

	res, err := http.Get(uri)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, NotFoundErr
	}

	dec := json.NewDecoder(res.Body)
	target := []*Dataset{}
	err = dec.Decode(&target)

	if err != nil {
		return nil, err
	}

	return target[0], nil
}

type dbentry struct {
	Age     time.Time
	Dataset *Dataset
}

type DB struct {
	fetcher StatFetcher
	cache   map[string]*dbentry
	mu      sync.RWMutex
}

func NewDB(fetcher StatFetcher) *DB {
	db := &DB{
		cache:   make(map[string]*dbentry),
		fetcher: fetcher,
	}
	go db.invalidate()
	return db
}

func (db *DB) invalidate() {
	c := time.Tick(time.Second * 30)
	expire := time.Duration(time.Minute * 10)

	for range c {
		db.mu.Lock()
		for k, v := range db.cache {
			if time.Since(v.Age) > expire {
				delete(db.cache, k)
			}
		}
		db.mu.Unlock()
	}
}

func (db *DB) key(name string, tf Timeframe) string {
	return name + tf.String()
}

func (db *DB) fetchRemote(name string, tf Timeframe) (*Dataset, error) {
	data, err := db.fetcher.Get(name, tf)
	if err != nil {
		return nil, err
	}
	db.mu.Lock()
	db.cache[db.key(name, tf)] = &dbentry{
		Age:     time.Now().UTC(),
		Dataset: data,
	}
	db.mu.Unlock()
	return data, nil
}

func (db *DB) Get(name string, tf Timeframe) (*Dataset, error) {
	db.mu.Lock()
	data, ok := db.cache[db.key(name, tf)]
	db.mu.Unlock()

	if ok {
		return data.Dataset, nil
	}

	return db.fetchRemote(name, tf)
}

type Server struct {
	db *DB
}

func NewServer(accessToken string) *Server {
	srv := &Server{
		db: NewDB(NewStatHatFetcher(accessToken)),
	}

	srv.initRoutes()
	return srv
}

func (srv *Server) initRoutes() {
	router := httprouter.New()
	router.GET("/v1/exportstats/:stat", srv.Index)

	// global middleware
	var middleware []func(http.Handler) http.Handler

	switch log.Severity {
	case log.LevelDebug:
		middleware = append(middleware, nocacheHandler, handler.LogHandler, handler.MeasureHandler, handler.DebugHandle, handler.RecoveryHandler)
	case log.LevelInfo:
		middleware = append(middleware, nocacheHandler, handler.LogHandler, handler.RecoveryHandler)
	default:
		middleware = append(middleware, nocacheHandler, handler.RecoveryHandler)
	}

	wrapped := handler.Use(router, middleware...)
	http.Handle("/", wrapped)
}

func (srv *Server) Index(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	stat := p.ByName("stat")
	log.Println(p.ByName("stat"), r.FormValue("t"))
	var tf Timeframe

	if t := r.FormValue("t"); t != "" {
		_tf, err := ParseTimeframe(t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tf = _tf
	} else {
		tf = MustParseTimeframe("1 hour @ 1 minute")
	}

	if start := r.FormValue("start"); start != "" {
		s, err := strconv.Atoi(start)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		t := time.Unix(int64(s), 0)
		tf.Start = &t
	}

	data, err := srv.db.Get(stat, tf)

	if err != nil {
		log.Error(err)
		if err == NotFoundErr {
			http.Error(w, "Not Found: "+stat, http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	format := r.FormValue("format")
	log.Println("count", len(data.Points))

	switch format {
	case "csv":
		csvWriter(w, data)
	case "json":
		jsonWriter(w, data)
	default:
		fmt.Fprint(w, data)
	}
}

func jsonWriter(w http.ResponseWriter, data *Dataset) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	jsonw := json.NewEncoder(w)
	err := jsonw.Encode(data.Points)
	if err != nil {
		log.Fatal(err)
	}
}

func csvWriter(w http.ResponseWriter, data *Dataset) {
	//w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	csvw := csv.NewWriter(w)

	for _, p := range data.Points {
		if err := csvw.Write(p.ToCSV()); err != nil {
			log.Fatal(err)
		}
	}

	csvw.Flush()

	if err := csvw.Error(); err != nil {
		log.Fatal(err)
	}
}

func nocacheHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		h.ServeHTTP(w, r)
	})
}
