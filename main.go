package main

import (
	"crypto/tls"
	"flag"
	"html/template"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/indigo-web/indigo"
	"github.com/indigo-web/indigo/http"
	"github.com/indigo-web/indigo/http/codec"
	"github.com/indigo-web/indigo/http/mime"
	"github.com/indigo-web/indigo/router/inbuilt"
	"github.com/indigo-web/indigo/router/inbuilt/middleware"
)

const (
	homeTemplate    = "templates/index.html"
	homeDefaultName = "Паша"
)

var (
	httpAddr  = flag.String("http", ":8080", "plain HTTP listen address")
	httpsAddr = flag.String("https", ":8443", "secure HTTP listen address")
	certdir   = flag.String("certdir", "", "TLS certificate directory")
)

func main() {
	flag.Parse()

	tmpl, err := template.ParseFiles(homeTemplate)
	if err != nil {
		log.Fatalf("cannot load home template: %s", err)
		return
	}

	r := inbuilt.New().
		Use(middleware.Recover).
		Use(middleware.LogRequests()).
		Get("/", func(request *http.Request) *http.Response {
			name := request.Params.ValueOr("n", homeDefaultName)
			resp := request.Respond()

			if err = tmpl.Execute(resp, name); err != nil {
				return http.Error(request, err)
			}

			return resp
		}).
		Get("/n/:name", func(request *http.Request) *http.Response {
			resp := request.Respond()
			name, err := url.PathUnescape(request.Vars.Value("name"))
			if err != nil {
				return http.Error(request, err)
			}
			if err = tmpl.Execute(resp, name); err != nil {
				return http.Error(request, err)
			}

			return resp
		}).
		Static("/static", "static", func(next inbuilt.Handler, request *http.Request) *http.Response {
			resp := next(request)

			if strings.HasSuffix(request.Path, ".css") {
				resp = resp.ContentType(mime.CSS)
			}

			return resp
		}).
		Alias("/age", "/static/age.html")

	app := indigo.New(*httpAddr)
	if len(*certdir) > 0 {
		certfile := path.Join(*certdir, "fullchain.pem")
		keyfile := path.Join(*certdir, "privkey.pem")
		cert, err := tls.LoadX509KeyPair(certfile, keyfile)
		if err != nil {
			log.Fatalf("failed to load TLS certficates: %s", err)
			return
		}

		app.TLS(*httpsAddr, cert)
	}

	log.Fatal(
		app.
			Codec(codec.Suit()...).
			OnBind(func(addr string) {
				log.Println("listening on", addr)
			}).
			Serve(r),
	)
}
