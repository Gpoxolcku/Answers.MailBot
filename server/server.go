package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"../workers"
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
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

func runParser(wr http.ResponseWriter, req *http.Request) {
	handler := func() workers.Result {

		response := workers.Result{}

		spec := strings.Join(reference['#'], "+")
		gwResp, err := http.Get("https://stackoverflow.com/search?q=" + spec)
		// gwResp, err := http.Get("https://stackoverflow.com/questions/24147059/how-do-i-format-a-string-into-edittext-in-android-with-aaa-aaa-aaa-format")
		if err != nil {
			log.Println(err)
			response.Code = http.StatusInternalServerError
			return response
		}

		defer gwResp.Body.Close()
		doc, err := goquery.NewDocumentFromReader(io.Reader(gwResp.Body))
		if err != nil {
			log.Println(err)
			response.Code = http.StatusBadGateway
			return response
		}

		doc.Find(".container").Find(".content").Find(".mainbar").
			Find(".question-summary search-result").Each(func(i int, s *goquery.Selection) { //тельце вопросика
			q := workers.Question{}
			q.Title, _ = s.Find(".summary").Find(".result-link").Attr("title")
			q.Link, _ = s.Find(".summary").Find(".result-link").Attr("href")
			// q.Tit
			response.QuestionList = append(response.QuestionList, q)
			println(s.Text())

		})

		response.Code = http.StatusOK

		return response
	}

	result, err := pool.AddTaskSyncTimed(handler, time.Second*5)
	if err != nil {
		log.Println(err)
	}
	out, err := yaml.Marshal(result)
	// out, err := yaml.Marshal("a\\nb\\nc")
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)
	}
	wr.WriteHeader(result.Code)
	wr.Write(out)

}

func parseQuestion(wr http.ResponseWriter, req *http.Request) { //сделать структуру: заголовок, первое предложение из статьи, ссылка

	log.Printf("Message /run/question received\n")

	handler := func() workers.Result {

		response := workers.Result{}

		// spec := strings.Join(reference['#'], "+")
		// response, err := http.Get("https://stackoverflow.com/search?q=" + spec)
		gwResp, err := http.Get("https://stackoverflow.com/questions/24147059/how-do-i-format-a-string-into-edittext-in-android-with-aaa-aaa-aaa-format")
		if err != nil {
			log.Println(err)
			response.Code = http.StatusInternalServerError
			return response
		}

		defer gwResp.Body.Close()
		doc, err := goquery.NewDocumentFromReader(io.Reader(gwResp.Body))
		if err != nil {
			log.Println(err)
			response.Code = http.StatusBadGateway
			return response
		}

		doc.Find(".postcell").Find(".post-text").Each(func(i int, s *goquery.Selection) { //тельце вопросика
			response.Text = strings.Join(strings.Split(s.Text(), "\\n"), "\n")
			println(s.Text())
			println(response.Text)

		})

		// doc.Find(".postcell").Find(".post-taglist").Each(func(i int, s *goquery.Selection) { //тэги вопросика
		// 	response.Tags = strings.Split(s.Text(), " ")
		// 	fmt.Print(s.Text())
		//
		// })

		doc.Find(".owner").Find(".user-info").Find(".user-details").Each(func(i int, s *goquery.Selection) { //ник и репутация спрашивающего
			response.NameOwner = s.Find("a").Text() //ник
			txt := strings.Join(strings.Split(s.Find(".reputation-score").Text(), ","), "")
			// fmt.Println("txt == " + txt)
			// *(response.ReputationOwner) = 10
			response.ReputationOwner = new(int)
			*(response.ReputationOwner), _ = strconv.Atoi(txt) //репутация

		})
		doc.Find(".answer").Each(func(i int, s *goquery.Selection) {
			attr, _ := s.Attr("data-answerid")
			response.IDAnswer = new(int)
			response.LikesAnswer = new(int)
			response.ReputationAnswer = new(int)
			*(response.IDAnswer), _ = strconv.Atoi(attr)                                                                                                         //id ответика
			*(response.LikesAnswer), _ = strconv.Atoi(s.Find(".vote-count-post ").Text())                                                                        //его рейтинг (лайки)
			*(response.ReputationAnswer), _ = strconv.Atoi(s.Find(".answercell").Find(".post-signature").Find(".user-details").Find(".reputation-score").Text()) //его репутация
			response.BodyAnswer = s.Find(".answercell").Find(".post-text").Text()                                                                                //его тельце
			response.NameAnswer = s.Find(".answercell").Find(".post-signature").Find(".user-details").Find("a").Text()                                           //имя отвечающего

		})

		response.Code = http.StatusOK
		return response

	}

	result, err := pool.AddTaskSyncTimed(handler, time.Second*5)
	if err != nil {
		log.Println(err)
	}
	out, err := yaml.Marshal(result)
	// out, err := yaml.Marshal("a\\nb\\nc")
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)
	}
	wr.WriteHeader(result.Code)
	wr.Write(out)

}

func editTopic(wr http.ResponseWriter, req *http.Request) {
	log.Printf("Message /topic/ received:\n")

	handler := func() workers.Result {

		response := workers.Result{}
		body, err := ioutil.ReadAll(req.Body)

		if err != nil {
			log.Println(err)
			response.Code = http.StatusInternalServerError
			return response
		}
		defer req.Body.Close()
		log.Printf("Message received:\n%s\n", body)

		topic = string(body)
		response.Code = http.StatusOK
		return response
	}

	result, err := pool.AddTaskSyncTimed(handler, time.Second)
	if err != nil {
		log.Println(err)
	}
	wr.WriteHeader(result.Code)
	// wr.Write([]byte(result.Response))

}

func editWordsDelete(wr http.ResponseWriter, req *http.Request) {

	fmt.Printf("Message /words.delete/ received\n")

	handler := func() workers.Result {
		response := workers.Result{}
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		log.Printf("Message received:\n%s\n", body)

		if err != nil {
			log.Println(err)
			response.Code = http.StatusInternalServerError
			return response
		}

		words := strings.Split(string(body), " ")
		delWords(reference, words)
		response.Code = http.StatusOK
		return response
	}

	result, err := pool.AddTaskSyncTimed(handler, time.Second)
	if err != nil {
		log.Println(err)
	}
	wr.WriteHeader(result.Code)
	// wr.Write([]byte(result.Response))

}

func editWordsAdd(wr http.ResponseWriter, req *http.Request) {

	// body, _ := ioutil.ReadAll(req.Body)
	// req.Body.Close()
	log.Printf("Message /words.add/ received\n")

	handler := func() workers.Result {

		response := workers.Result{}
		body, err := ioutil.ReadAll(req.Body)
		// head := req.Header
		req.Body.Close()
		fmt.Printf("Message received:\n%s\n", body)

		if err != nil {
			log.Println(err)
			response.Code = http.StatusInternalServerError
			return response
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
		response.Code = http.StatusOK
		return response

	}

	result, err := pool.AddTaskSyncTimed(handler, time.Second)
	if err != nil {
		log.Println(err)
	}
	wr.WriteHeader(result.Code)
	// wr.Write([]byte(result.Response))

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
