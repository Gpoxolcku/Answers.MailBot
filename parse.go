package parser

import (
	"io"
	"net/http"
	"os"
	"github.com/PuerkitoBio/goquery"
)

func CheckError(err error){
	if err != nil {
			   panic(err)
			   os.Exit(1)
			}
}

func main(){
	//spec := strings.Join(os.Args[1:], "+")
	//response, err := http.Get("https://stackoverflow.com/search?q=" + spec)
	response, err:=http.Get("http://lk.fcsm.ru/Home/Outgoing/backoff%40qbfin.ru")
	CheckError(err)
	defer response.Body.Close()		
	doc,err:=goquery.NewDocumentFromReader(io.Reader(response.Body))
	CheckError(err)
	doc.Find(".postcell").Find(".post-text").Each(func(i int, s *goquery.Selection){   //тельце вопросика  
		println(s.Text())
	})
	doc.Find(".postcell").Find(".post-taglist").Each(func(i int, s *goquery.Selection){   //тэги вопросика
		println(s.Text())
	})

	doc.Find(".owner").Find(".user-info"). Find(".user-details").Each(func(i int, s *goquery.Selection){ //ник и репутация спрашивающего
		nameOwner:=s.Find("a") //ник
		reputationOwner:=s.Find(".reputation-score") //репутация
		println(nameOwner.Text())
		println(reputationOwner.Text())
		})
	doc.Find(".answer").Each(func(i int, s *goquery.Selection){
		idAnswer,_:=s.Attr("data-answerid") //id ответика
		likesAnswer:=s.Find(".vote-count-post ") //его рейтинг (лайки)
		bodyAnswer:=s.Find(".answercell").Find(".post-text") //его тельце
		nameAnswer:=s.Find(".answercell").Find(".post-signature").Find(".user-details").Find("a") //имя отвечающего
		reputationAnswer:=s.Find(".answercell").Find(".post-signature").Find(".user-details").Find(".reputation-score") //его репутация
		println(idAnswer)
		println(likesAnswer.Text())
		println(bodyAnswer.Text())
		println(nameAnswer.Text())
		println(reputationAnswer.Text())
	})

}
