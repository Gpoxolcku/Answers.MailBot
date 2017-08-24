package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"../workers"
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
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

	handler := func() int {

		response, err := http.Get("http://lk.fcsm.ru/Home/Outgoing/backoff%40qbfin.ru")
		if err != nil {
			log.Println(err)
			return http.StatusBadRequest
		}

		defer response.Body.Close()
		doc, err := goquery.NewDocumentFromReader(io.Reader(response.Body))
		if err != nil {
			log.Println(err)
			return http.StatusBadGateway
		}

		doc.Find(".postcell").Find(".post-text").Each(func(i int, s *goquery.Selection) { //тельце вопросика
			println(s.Text())
		})

		doc.Find(".postcell").Find(".post-taglist").Each(func(i int, s *goquery.Selection) { //тэги вопросика
			println(s.Text())
		})

		doc.Find(".owner").Find(".user-info").Find(".user-details").Each(func(i int, s *goquery.Selection) { //ник и репутация спрашивающего
			nameOwner := s.Find("a")                       //ник
			reputationOwner := s.Find(".reputation-score") //репутация
			println(nameOwner.Text())
			println(reputationOwner.Text())
		})

		doc.Find(".answer").Each(func(i int, s *goquery.Selection) {
			idAnswer, _ := s.Attr("data-answerid")                                                                            //id ответика
			likesAnswer := s.Find(".vote-count-post ")                                                                        //его рейтинг (лайки)
			bodyAnswer := s.Find(".answercell").Find(".post-text")                                                            //его тельце
			nameAnswer := s.Find(".answercell").Find(".post-signature").Find(".user-details").Find("a")                       //имя отвечающего
			reputationAnswer := s.Find(".answercell").Find(".post-signature").Find(".user-details").Find(".reputation-score") //его репутация

			println(idAnswer)
			println(likesAnswer.Text())
			println(bodyAnswer.Text())
			println(nameAnswer.Text())
			println(reputationAnswer.Text())

		})

		return http.StatusOK

	}

	status, err := pool.AddTaskSyncTimed(handler, time.Second)
	if err != nil {
		log.Println(err)
	}
	wr.WriteHeader(status)

}

func editTopic(wr http.ResponseWriter, req *http.Request) {
	log.Printf("Message /topic/ received:\n")

	handler := func() int {
		body, err := ioutil.ReadAll(req.Body)
		defer req.Body.Close()
		log.Printf("Message received:\n%s\n", body)

		if err != nil {
			log.Println(err)
			return http.StatusInternalServerError
		}

		topic = string(body)
		return http.StatusOK
	}

	status, err := pool.AddTaskSyncTimed(handler, time.Second)
	if err != nil {
		log.Println(err)
	}
	wr.WriteHeader(status)

}

func editWordsDelete(wr http.ResponseWriter, req *http.Request) {

	fmt.Printf("Message /words.delete/ received\n")

	handler := func() int {
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		log.Printf("Message received:\n%s\n", body)

		if err != nil {
			log.Println(err)
			return http.StatusInternalServerError
		}

		words := strings.Split(string(body), " ")
		delWords(reference, words)
		return http.StatusOK
	}

	status, err := pool.AddTaskSyncTimed(handler, time.Second)
	if err != nil {
		log.Println(err)
	}
	wr.WriteHeader(status)

}

func editWordsAdd(wr http.ResponseWriter, req *http.Request) {

	// body, _ := ioutil.ReadAll(req.Body)
	// req.Body.Close()
	log.Printf("Message /words.add/ received\n")

	handler := func() int {

		body, err := ioutil.ReadAll(req.Body)
		// head := req.Header
		req.Body.Close()
		fmt.Printf("Message received:\n%s\n", body)

		if err != nil {
			log.Println(err)
			return http.StatusInternalServerError
		}

		words := strings.Split(string(body), " ")
		addWords(reference, words)

		// for first, words := range reference {
		// 	fmt.Printf("list %q:\n", first)
		// 	for _, word := range words {
		// 		fmt.Printf("%s, ", word)
		// 	}
		// 	fmt.Printf("\n\n")
		// }
		return http.StatusOK

	}

	status, err := pool.AddTaskSyncTimed(handler, time.Second)
	if err != nil {
		log.Println(err)
	}
	wr.WriteHeader(status)

}

// TODO: Нужно сделать для каждого клиента!!
var reference map[rune][]string
var topic string
var spec struct {
	SortBy int `json:"sort_by"`
}

// Start a server with @parameter concurrency pool size
func Start(concurrency int, addr string) {

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
	pool = workers.NewPool(concurrency)
	pool.Run()

	log.Fatal(http.ListenAndServe(addr, router))
}
