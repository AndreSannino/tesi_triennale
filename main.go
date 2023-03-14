package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"text/template"
	"time"
)

/*-------JSON OBJECT--------*/

type BaseReading struct {
	Id           string `json:"id"`
	Origin       int    `json:"origin"`
	DeviceName   string `json:"deviceName"`
	ResourceName string `json:"resourceName"`
	ProfileName  string `json:"profileName"`
	ValueType    string `json:"valueType"`
	Value        string `json:"value"`
}

type MultiReadingsResponse struct {
	ApiVersion string        `json:"apiVersion"`
	RequestId  string        `json:"requestId"`
	Message    string        `json:"message"`
	StatusCode int           `json:"statusCode"`
	TotalCount int           `json:"totalCount"`
	Readings   []BaseReading `json:"readings"`
}

/*----------------------------------*/

/*Singola lettura*/
type reading struct {
	Value     int
	timestamp int
}

/*Struttura della risorsa condivisa*/
type sensorData struct {
	mutexl      sync.Mutex
	sync        sync.Mutex
	num_request int
	dim         int
	Data        []reading
	LastUpdate  string
}

func main() {
	var exit string
	var sd sensorData

	//Inizializzo il vettore con dimensione pari a 100
	sd.Init(100)

	//Avvio la routine per aggiornare i valori
	go dataAcquisition(&sd)

	/*Utilizzo del ServeMux di default di GO. Aggiungo la funzione HandleRequest per gestire l'indirizzo radice*/
	http.HandleFunc("/", sd.HandleRequest)

	/*Avvio la routine che mette il server in ascolto sulla porta 8080*/
	go http.ListenAndServe(":8080", nil)

	fmt.Println("Digita invio per uscire")
	fmt.Scanln(&exit)
	fmt.Println("Uscita...")
}

func dataAcquisition(s *sensorData) {
	var data MultiReadingsResponse
	coreDataApiAddress := "http://localhost:59880/api/v2/reading/device/name/Random-Integer-Device/resourceName/Int8?&limit="
	var httpRequest string = coreDataApiAddress + strconv.Itoa(s.dim) //compongo al richiesta API
	var r reading                                                     //elemento di appoggio che verrà inserito nel vettore condiviso
	var i int
	loc, _ := time.LoadLocation("Europe/Rome")
	for {
		httpResponse, err := http.Get(httpRequest)

		if err != nil {
			log.Fatal(err)
		}

		rawResponseData, err2 := ioutil.ReadAll(httpResponse.Body) //leggo la risposta
		if err2 != nil {
			log.Fatal(err2)
		}

		err = json.Unmarshal(rawResponseData, &data) //l'oggetto JSON di risposta viene messo nella variabile di appoggio data
		if err != nil {
			log.Fatal(err)
		}
		i = 0
		s.startWriting() //ingresso zona critica
		for _, read := range data.Readings {
			r.Value, err = strconv.Atoi(read.Value)
			if err != nil {
				log.Fatal(err)
			}
			r.timestamp = read.Origin
			s.Data[i] = r //push dell'elemento nel vettore
			i++
		}

		/*L'elemento che si trova si trova nella prima posizione è quello più recente*/
		s.LastUpdate = time.Unix(0, int64(data.Readings[0].Origin)).In(loc).Format(time.DateTime)

		/*Ordinamento array in ordine cronologico*/
		sort.Slice(s.Data, func(i int, j int) bool { return s.Data[i].timestamp < s.Data[j].timestamp })
		s.finishWriting() //uscita zona critica
		time.Sleep(2 * time.Second)
	}
}

func (s *sensorData) startReading() {
	s.mutexl.Lock()
	s.num_request++
	if s.num_request == 1 {
		s.sync.Lock()
	}
	s.mutexl.Unlock()
}

func (s *sensorData) finishReading() {
	s.mutexl.Lock()
	s.num_request--
	if s.num_request == 0 {
		s.sync.Unlock()
	}
	s.mutexl.Unlock()
}

func (s *sensorData) startWriting() {
	s.sync.Lock()
}

func (s *sensorData) finishWriting() {
	s.sync.Unlock()
}

func (s *sensorData) HandleRequest(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("index.html")
	if err != nil {
		log.Fatal(err)
	}
	s.startReading()
	t.Execute(w, s)
	s.finishReading()
}

func (sd *sensorData) Init(dim int) {
	sd.Data = make([]reading, dim)
	sd.dim = dim
}
