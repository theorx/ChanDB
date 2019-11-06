package main

import (
	"log"
	"net/http"
)

func main() {

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		//payload, _ := json.Marshal(struct{ Status string `json:"status"` }{Status: "All bueno mofo"})
		writer.Write([]byte("Dantini"))
	})

	log.Fatalln(http.ListenAndServe(":8080", nil))

}
