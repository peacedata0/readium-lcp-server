package api

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/jpbougie/lcpserve/epub"
	"github.com/jpbougie/lcpserve/index"
	"github.com/jpbougie/lcpserve/pack"
	"github.com/jpbougie/lcpserve/storage"

	"io/ioutil"
	"log"
	"net/http"
)

type Server interface {
	Store() storage.Store
	Index() index.Index
	Certificate() *tls.Certificate
}

func StorePackage(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	name := vars["name"]
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body")
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	sha := sha256.Sum256(buf)
	key := fmt.Sprintf("%x", sha)

	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		log.Println("Error opening zip")
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	ep, err := epub.Read(zr)
	if err != nil {
		log.Println("Error reading epub")
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	out, encryptionKey, err := pack.Do(ep)
	if err != nil {
		log.Println("Error packing")
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	output := new(bytes.Buffer)
	out.Write(output)
	_, err = s.Store().Add(key, output)
	if err != nil {
		log.Println("Error storing")
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	err = s.Index().Add(index.Package{key, encryptionKey, name})
	if err != nil {
		log.Println("Error while adding to index")
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte(name))
}

func ListPackages(w http.ResponseWriter, r *http.Request, s Server) {
	fn := s.Index().List()
	packages := make([]index.Package, 0)

	for it, err := fn(); err == nil; it, err = fn() {
		packages = append(packages, it)
	}

	enc := json.NewEncoder(w)
	err := enc.Encode(packages)
	if err != nil {
		w.WriteHeader(500)
	}

}