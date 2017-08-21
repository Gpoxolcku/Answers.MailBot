package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"../workers"
	"github.com/gorilla/mux"
	"github.com/moovweb/gokogiri"
)

// type IPool interface {
// 	Size() int
// 	Run()
// 	AddTaskSyncTimed(f workers.Func, timeout time.Duration) (interface{}, error)
// }

// var pool *workers.Pool = workers.NewPool(5)

var pool *workers.Pool

func contains(s []string, e string) (bool, int) {
	for i, a := range s {
		if a == e {
			return true, i
		}
	}
	return false, -1
}

func checkWordInLists(check map[rune][]string, first rune, word string) bool {

	alias := map[rune]string{
		'-': "exclude words",
		'+': "essential words",
		'#': "keywords",
	}

	for key, valmap := range check {
		ok, _ := contains(valmap, word)
		if ok {
			log.Printf("%s: Word is in %s already.\n", string(word), alias[key])
			return true
		}
	}
	return false
}

func parseWord(str string) (first rune, word string) {
	runes := []rune(str)
	first = runes[0]
	// Смотрим на первый символ ключевого слова
	switch first {
	case '-':
		word = string(runes[1:])
	case '!':
		word = string(runes[1:])
	default:
		// Обычное ключевое слово, обозначим через '#'
		first = '#'
		word = string(runes[:])
	}

	return first, word
}

func addWords(reference map[rune][]string, words []string) {

	for _, r := range words {

		first, word := parseWord(r)
		f := checkWordInLists(reference, first, word)
		if f {
			log.Printf("Keyword error:\n")
			continue
		}
		_, ok := reference[first]
		if !ok {
			reference[first] = []string{}
		}
		reference[first] = append(reference[first], word)
	}
}

func delWords(reference map[rune][]string, words []string) {

	alias := map[rune]string{
		'-': "exclude words",
		'+': "essential words",
		'#': "keywords",
	}

	for _, r := range words {

		first, word := parseWord(r)
		// _, ok := reference[first][string(word)]
		// if !ok {
		// 	log.Fatalf("Keyword error:\n%s: No such word in %s\n", string(word), alias[first])
		// }
		ok, i := contains(reference[first], word)
		if ok {
			reference[first] = append(reference[first][:i], reference[first][i+1:]...)
		} else {
			log.Printf("Keyword error:\n%s: No such word in %s\n", string(word), alias[first])
		}
	}
}

func runParser(wr http.ResponseWriter, req *http.Request) { //сделать структуру: заголовок, первое предложение из статьи, ссылка

	log.Printf("Message /run/ received\n")

	handler := func() interface{} {

		res, err := http.Get("http://otvet.mail.ru/search/" + strings.Join(reference['#'], " "))
		if err != nil {
			log.Fatal(err)
		}
		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		if err != nil {
			log.Fatal(err)
		}

		// log.Printf("Message received:\n%s\n", body)
		doc, err := gokogiri.ParseHtml(body)
		fl, err := os.Create("file.txt")
		// fl, err := os.Open("file.txt")
		defer fl.Close()
		fmt.Fprintf(fl, "%s\n", doc.Content())
		if err != nil {
			log.Fatal(err)
		}
		xp := "//*[@id=\"ColumnCenter\"]/div[2]/div/div[3]/div"
		nodes, err := doc.Search(xp)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%s:\nnum of matching nodes = %d\n", xp, len(nodes))

		// node[0].CountChildren()

		// panelPath := "[@id=\"ColumnCenter\"]/div[2]/div/div[3]/div"

		return nil

	}

	pool.AddTaskSyncTimed(handler, time.Second)
	wr.WriteHeader(http.StatusOK)

}

func editTopic(wr http.ResponseWriter, req *http.Request) {
	log.Printf("Message /topic/ received:\n")

	handler := func() interface{} {
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		log.Printf("Message received:\n%s\n", body)

		if err != nil {
			log.Fatal(err)
		}

		topic = string(body)
		return nil
	}

	pool.AddTaskSyncTimed(handler, time.Second)
	wr.WriteHeader(http.StatusOK)

}

func editWordsDelete(wr http.ResponseWriter, req *http.Request) {

	fmt.Printf("Message /words.delete/ received\n")

	handler := func() interface{} {
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		log.Printf("Message received:\n%s\n", body)

		if err != nil {
			log.Fatal(err)
		}

		words := strings.Split(string(body), " ")
		delWords(reference, words)
		return nil
	}

	pool.AddTaskSyncTimed(handler, time.Second)
	wr.WriteHeader(http.StatusOK)

}

func editWordsAdd(wr http.ResponseWriter, req *http.Request) {

	// body, _ := ioutil.ReadAll(req.Body)
	// req.Body.Close()
	log.Printf("Message /words.add/ received\n")

	handler := func() interface{} {

		/*		commandsList := []string{
					"start",
					"stop",
					"topic",
					"add",
					"del",
					"sort",
				}

				argsLenDict := map[string]int{
					"start": 0,
					"stop":  0,
					"topic": 1,
					"add":   -1,
					"del":   -1,
					"sort":  1,
				}
		*/
		body, err := ioutil.ReadAll(req.Body)
		// head := req.Header
		req.Body.Close()
		fmt.Printf("Message received:\n%s\n", body)

		if err != nil {
			log.Fatal(err)
		}

		words := strings.Split(string(body), " ")
		addWords(reference, words)

		//if len(str) > 15 {
		//	panic("Too much arguments")
		//}

		/*	for i, r := range str[2:] {
			    parse_arg(command, r, i)
			}
		*/

		for first, words := range reference {
			fmt.Printf("list %q:\n", first)
			for _, word := range words {
				fmt.Printf("%s, ", word)
			}
			fmt.Printf("\n\n")
		}
		return nil

	}

	pool.AddTaskSyncTimed(handler, time.Second)
	wr.WriteHeader(http.StatusOK)

}

// TODO: Нужно сделать для каждого клиента!!
var reference map[rune][]string
var topic string
var spec struct {
	SortBy int `json:"sort_by"`
}

// Start a server with @parameter concurency pool size
func Start(concurency int, addr string) {

	// const maxArgs = 10
	//
	// if len(str) < 1 {
	// 	fmt.Printf("help")
	// 	return
	// }
	reference = make(map[rune][]string)
	router := mux.NewRouter()
	router.HandleFunc("/words", editWordsAdd).Methods("POST")
	router.HandleFunc("/words", editWordsDelete).Methods("DELETE")
	router.HandleFunc("/topic", editTopic).Methods("POST")
	// router.HandleFunc("/spec", editSpec).Methods("POST")
	router.HandleFunc("/run", runParser).Methods("GET")
	pool = workers.NewPool(concurency)
	pool.Run()

	log.Fatal(http.ListenAndServe(addr, router))
}
