package privateNotes

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/storage"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	// Register an HTTP function with the Functions Framework
	functions.HTTP("privateNotes", privateNotes)
}

type SecretNote struct {
	Key        string
	SecureNote string
}

type IndexPageData struct {
	PostUrl string
}

type SuccessPageData struct {
	SecretUrl string
}

func privateNotes(w http.ResponseWriter, r *http.Request) {

	// os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./key.json") // for local development
	function_path := "./serverless_function_source_code/"
	// function_path := "./" // for local development
	GCP_PROJECT := os.Getenv("GCP_PROJECT")
	GCP_REGION := os.Getenv("GCP_REGION")
	GCP_CF_NAME := os.Getenv("GCP_CF_NAME")
	switch r.Method {
	case http.MethodGet:
		key := r.URL.Query().Get("key")
		if key != "" {
			ctx := context.Background()
			client, err := storage.NewClient(ctx)
			if err != nil {
				fmt.Println("Error: ", err)
			}
			rc, err := client.Bucket("private-notes").Object(key).NewReader(ctx)
			if err != nil {
				fmt.Println("Error: ", err)
				// http.Error(w, "Note does not exist", http.StatusNotFound)
				tmpl := template.Must(template.ParseFiles(function_path + "templates/error.html"))
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				tmpl.Execute(w, "")
				return
			}
			// defer
			slurp, err := ioutil.ReadAll(rc)
			rc.Close()
			if err != nil {
				fmt.Println("Error: ", err)
				return
			}
			fmt.Println(string(slurp))

			if err := client.Bucket("private-notes").Object(key).Delete(ctx); err != nil {
				fmt.Println("Error: ", err)
			}

			data := SecretNote{
				Key:        key,
				SecureNote: string(slurp),
			}
			tmpl := template.Must(template.ParseFiles(function_path + "templates/result.html"))
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			tmpl.Execute(w, data)
			return
		} else {
			log.Printf("get")
			data := IndexPageData{
				PostUrl: GCP_CF_NAME,
			}
			tmpl := template.Must(template.ParseFiles(function_path + "templates/index.html"))
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			tmpl.Execute(w, data)
			return
		}
	case http.MethodPost:
		log.Printf("post")
		// ##################### Get the form data
		r.ParseForm()
		var t SecretNote
		t.Key = r.FormValue("key")
		t.SecureNote = r.FormValue("secureNote")
		// log.Println(t.Test)
		log.Println(t.Key)
		log.Println(t.SecureNote)
		// ##################### Prepare the url
		data := SuccessPageData{
			SecretUrl: string(GCP_REGION + "-" + GCP_PROJECT + ".cloudfunctions.net/" + GCP_CF_NAME + "?key=" + t.Key),
		}
		// ##################### Save the cipherText to bucket
		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		wc := client.Bucket("private-notes").Object(t.Key).NewWriter(ctx)
		wc.ContentType = "text/plain"
		// wc.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
		if _, err := wc.Write([]byte(t.SecureNote)); err != nil {
			// TODO: handle error.
			// Note that Write may return nil in some error situations,
			// so always check the error from Close.
			fmt.Println("Error: ", err)
		}
		if err := wc.Close(); err != nil {
			fmt.Println("Error: ", err)
		}

		// ##################### Render the reponse template
		tmpl := template.Must(template.ParseFiles(function_path + "templates/success.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, data)
		return
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}