package main

import (
	"github.com/russross/blackfriday"
	"html/template"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"regexp"
)

const lenPath = len("/view/")
const tmplPath = "tmpl/"

//const dataPath = "articles/"

// Regexp for titles
var titleValidator = regexp.MustCompile("^[a-zA-Z0-9]+$")

// Templates caching
var templates = template.Must(template.ParseFiles(tmplPath+"header.html", tmplPath+"footer.html", tmplPath+"index.html", tmplPath+"edit.html", tmplPath+"view.html"))

// Validation wrapper
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Path[lenPath:]
		if !titleValidator.MatchString(title) {
			http.NotFound(w, r)
			return
		}
		fn(w, r, title)
	}
}

// MongoDB
func mongoConnect() (*mgo.Collection, *mgo.Session, error) {
	// MGO connexion
	//session, err := mgo.Dial("127.0.0.1:27017")
	session, err := mgo.Dial("127.0.0.1:27017,paulo.mongohq.com:10075/go_wiki")
	if err != nil {
		return nil, nil, err
	}
	c := session.DB("go_wiki").C("articles")
	return c, session, nil
}

//  pages save
func (p *Page) save() error {
	c, s, err := mongoConnect()
	defer s.Close()
	if err != nil {
		return err
	}
	err = c.Insert(p)
	if err != nil {
		return err
	}
	return nil
}

// page load
func loadPage(pagetitle string) (*Page, error) {
	result := Page{}
	c, s, err := mongoConnect()
	defer s.Close()
	if err != nil {
		return nil, err
	}
	err = c.Find(bson.M{"title": pagetitle}).One(&result)
	if err != nil {
		return nil, err
	}
	output := blackfriday.MarkdownCommon(result.Body)
	return &Page{Title: pagetitle, Body: output}, nil
}

// page build
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	templates.ExecuteTemplate(w, "header.html", p)
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	templates.ExecuteTemplate(w, "footer.html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HANDLERS
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Redirect(w, r, "/view/"+title, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

// this function renders the home page, retrieving the number of articles then lists them
func indexHandler(w http.ResponseWriter, r *http.Request) {
	//result := Page{}
	c, s, err := mongoConnect()
	defer s.Close()
	if err != nil {
		return
	}
	nbArt, err := c.Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	//err = c.Find()

	templates.ExecuteTemplate(w, "header.html", nbArt)
	err = templates.ExecuteTemplate(w, "index.html", nbArt)
	templates.ExecuteTemplate(w, "footer.html", nbArt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

func main() {
	// static ressources
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	http.Handle("/font/", http.StripPrefix("/font/", http.FileServer(http.Dir("font"))))
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))

	// Handlers
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	// web magic
	http.ListenAndServe(":8080", nil)

}
