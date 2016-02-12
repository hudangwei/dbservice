package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
)

func getRequestParams(r *http.Request, urlParams map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}
	for k, v := range r.Form {
		if len(v) >= 1 {
			params[k] = v[0]
		}
	}
	if r.Header.Get("Content-Type") == "application/json" {
		decoder := json.NewDecoder(r.Body)
		requestBodyMap := make(map[string]interface{})
		err = decoder.Decode(&requestBodyMap)
		if err != nil {
			return nil, err
		}
		for k, v := range requestBodyMap {
			params[k] = v
		}
	}
	for k, v := range urlParams {
		params[k] = v
	}
	return params, nil
}

func handler(route *Route) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		urlParams := make(map[string]interface{})
		for _, urlParam := range ps {
			urlParams[urlParam.Key] = urlParam.Value
		}
		params, err := getRequestParams(r, urlParams)
		sql, err := route.Sql(params, 0)
		if err != nil && sql != "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, sql)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		log.Println(sql)
		rows, err := db.Query(sql)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		defer rows.Close()
		var jsonValue string
		for rows.Next() {
			err := rows.Scan(&jsonValue)
			if err != nil {
				if route.Collection {
					fmt.Fprint(w, "[]")
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
				return
			}
			fmt.Fprint(w, jsonValue)
		}

	}
}

var db *sql.DB

func main() {
	api, err := ParseRoutes(".")
	if err != nil {
		log.Fatal(err)
	}
	db, err = GetDbConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	router := httprouter.New()
	for _, route := range api.Routes {
		if route.Method == "GET" {
			router.GET(route.Path, handler(route))
		}
		if route.Method == "POST" {
			router.POST(route.Path, handler(route))
		}
		if route.Method == "PUT" {
			router.PUT(route.Path, handler(route))
		}
		if route.Method == "DELETE" {
			router.DELETE(route.Path, handler(route))
		}
	}
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	log.Fatal(http.ListenAndServe(":"+port, router))
}
