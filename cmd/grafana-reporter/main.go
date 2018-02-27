/*
   Copyright 2016 Vastech SA (PTY) LTD

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/IzakMarais/reporter"
	"github.com/IzakMarais/reporter/grafana"
)

var proto = flag.String("proto", "http://", "Grafana Protocol")
var ip = flag.String("ip", "localhost:3000", "Grafana IP and port")
//J var port = flag.String("port", ":8686", "Port to serve on")
// WARNING ** Port changed for parallel debug. Restore original for production **
var port = flag.String("port", ":8687", "Port to serve on") // DEBUG
var templateDir = flag.String("templates", "templates/", "Directory for custom TeX templates")

func main() {
	flag.Parse()
	log.SetOutput(os.Stdout)

	//'generated*'' variables injected from build.gradle: task 'injectGoVersion()'
	log.Printf("grafana reporter, version: %s.%s-%s hash: %s", generatedMajor, generatedMinor, generatedRelease, generatedGitHash)
	log.Printf("serving at '%s' and using grafana at '%s'", *port, *ip)

	router := mux.NewRouter()
	router.HandleFunc("/api/report/{dashName}", serveReport)

	log.Fatal(http.ListenAndServe(*port, router))
}

func serveReport(w http.ResponseWriter, req *http.Request) {
	log.Print("Reporter called")
	
	// Push HTTP Request to grafana package.
	grafana.GlobalReq = req
	
	g := grafana.NewClient(*proto+*ip, apiToken(req), dashVariable(req))
	rep := report.New(g, dashName(req), time(req), texTemplate(req))

	file, err := rep.Generate()
	if err != nil {
		log.Println("Error generating report:", err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer rep.Clean()
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		log.Println("Error copying data to response:", err)
		http.Error(w, err.Error(), 500)
		return
	}
	log.Println("Report generated correctly")
}

func dashName(r *http.Request) string {
	vars := mux.Vars(r)
	d := vars["dashName"]
	log.Println("Called with dashboard:", d)
	return d
}

func time(r *http.Request) grafana.TimeRange {
	params := r.URL.Query()
	t := grafana.NewTimeRange(params.Get("from"), params.Get("to"))
	log.Println("Called with time range:", t)
	return t
}

func apiToken(r *http.Request) string {
	apiToken := r.URL.Query().Get("apitoken")
	log.Println("Called with api Token:", apiToken)
	return apiToken
}

func dashVariable(r *http.Request) string {
	if strings.Contains(r.URL.RequestURI(), "var-") == true {
// Since we do not know the variable name, we search using Split on known key : &var-
		dashVariable := strings.Split(r.URL.RequestURI(), "var-")[1]
		dashVariable  = strings.Split(dashVariable, "&")[0]
		log.Println("Called with variable:", dashVariable)
		return dashVariable
	} else {
		log.Println("Called without variable")
		return ""
	}
}

func texTemplate(r *http.Request) string {
	fName := r.URL.Query().Get("template")
	if fName == "" {
		return ""
	}
	file := filepath.Join(*templateDir, fName+".tex")
	log.Println("Called with template:", file)

	customTemplate, err := ioutil.ReadFile(file)
	if err != nil {
		log.Printf("Error reading template file: %q", err)
		return ""
	}

	return string(customTemplate)
}
