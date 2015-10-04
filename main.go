package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gorilla/mux"
	"github.com/rwcarlsen/goexif/exif"
)

const storage = "/media/sf_Photos"
const cacheDir = "/media/sf_Photos/tmp"

type FileInfo struct {
	Name     string      `json:"name,omitempty"`
	Size     int64       `json:"size"`
	Mode     os.FileMode `json:"mode"`
	ModTime  time.Time   `json:"modTime"`
	IsDir    bool        `json:"isDir"`
	IsPhoto  bool        `json:"isPhoto"`
	Exif     exif.Exif   `json:"exif,omitempty"`
	Url      string      `json:"img"`
	ThumbUrl string      `json:"thumb"`
}

func readRotation(path string) int {
	ef, err := os.Open(path)
	if err != nil {
		log.Print(err)
		return 1
	}
	e, err := exif.Decode(ef)
	if err != nil {
		log.Print(err)
		return 1
	}
	o, err := e.Get(exif.Orientation)
	if err != nil {
		log.Print(err)
		return 1
	}
	i, err := o.Int(0)
	if err != nil {
		log.Print(err)
		return 1
	}
	return i
}

func thumbnailHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	height, err := strconv.Atoi(query.Get("height"))
	if err != nil {
		height = 50
	}
	path := filepath.Join(storage, r.URL.Path)
	photo, err := imaging.Open(path)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	switch readRotation(path) {
	case 8:
		photo = imaging.Rotate90(photo)
	case 2:
		photo = imaging.Rotate180(photo)
	case 6:
		photo = imaging.Rotate270(photo)
	}
	thumb := imaging.Resize(photo, 0, height, imaging.Box)
	imaging.Encode(w, thumb, imaging.JPEG)
}

func withJsonDirectoryListing(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			path := filepath.Join(storage, r.URL.Path)
			files, err := ioutil.ReadDir(path)
			if err != nil {
				log.Fatal(err)
			}
			list := []FileInfo{}
			for _, file := range files {
				f := FileInfo{
					Name:     file.Name(),
					Size:     file.Size(),
					ModTime:  file.ModTime(),
					IsDir:    file.IsDir(),
					IsPhoto:  !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".jpg"),
					Url:      (&url.URL{Path: "/photos/" + r.URL.Path + file.Name()}).String(),
					ThumbUrl: (&url.URL{Path: "/thumbnail/" + r.URL.Path + file.Name()}).String(),
				}
				if f.IsPhoto { // read exif
					ef, err := os.Open(filepath.Join(path, file.Name()))
					if err != nil {
						log.Print(err)
						continue
					}
					e, err := exif.Decode(ef)
					if err != nil {
						log.Print(err)
						continue
					}
					f.Exif = *e
				}
				list = append(list, f)
			}
			result, err := json.Marshal(list)
			if err != nil {
				log.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(result)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir("static")))
	r.Handle("/thumbnail/", http.StripPrefix("/thumbnail/", http.HandlerFunc(thumbnailHandler)))
	r.Handle("/photos/", http.StripPrefix("/photos/", withJsonDirectoryListing(http.FileServer(http.Dir(storage)))))
	http.Handle("/", r)
	http.ListenAndServe("0.0.0.0:8000", nil)
}
