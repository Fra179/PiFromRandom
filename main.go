package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"sync"
)

const (
	total              = 1e8
	// Il programma scarica i numeri da un sito che fornisce numeri random generati da generatori hardware
	url                = "https://qrng.anu.edu.au/API/jsonI.php?type=uint16&length="
	steps              = 20
	concurrentRequests = 2 * 100
	activationLimit    = 30
)

var (
	coprimes = 0
)

type Numbers struct {
	Success bool  `json:"success"`
	Data    []int `json:"data"`
}

// Algoritmo di Euclide per il calcolo del Massimo Comun Divisore
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}

	return a
}

// Stampa una progress bar per capire a che punto si trova il programma
func printProgress(actual, max int) {
	stepsMade := actual * steps / max
	fmt.Print("|")
	for x := 0; x < stepsMade-1; x++ {
		fmt.Print("=")
	}

	if stepsMade != 0 {
		fmt.Print(">")
	}

	for x := 0; x < steps-stepsMade; x++ {
		fmt.Print(" ")
	}

	fmt.Printf("| %d/%d\r", actual, max)
}

// Esegue la richiesta per i numeri random e salva il risultato in una struttura per poi ritornarlo
func makeNumsRequest(num int) []int {
	if num == 0 {
		return []int{}
	}

	var buf Numbers
	resp, _ := http.Get(url + fmt.Sprint(num))
	body, _ := ioutil.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &buf); err != nil {
		panic(err)
	}

	if !buf.Success {
		panic("request failed")
	}

	return buf.Data
}

// Essendo che ogni richiesta può ricevere solamente 1024 numeri random, se si
// vogliono ricevere più di 1024 numeri bisogna fare una suddivisione in "batch"
func getRandomDigits(num int, progress bool) []int {
	var nums []int
	maxRequests := num / 1024

	fmt.Println("Requesting true random numbers")

	for x := 0; x < maxRequests; x++ {
		n := makeNumsRequest(1024)
		nums = append(nums, n...)
		if progress {
			printProgress(x+1, maxRequests)
		}
	}

	n := makeNumsRequest(num % 1024)

	if progress {
		fmt.Println()
	}

	return append(nums, n...)
}

func getThreadedRandomDigits(num int) []int {
	var (
		completed      = 0
		completedMutex sync.Mutex
		mutex          sync.Mutex
		numbers        []int
	)

	if num > activationLimit {
		for x := 0; x < concurrentRequests; x++ {
			go func() {
				buf := getRandomDigits(num/concurrentRequests, false)

				mutex.Lock()
				numbers = append(numbers, buf...)
				mutex.Unlock()

				buf = nil

				completedMutex.Lock()
				completed++
				completedMutex.Unlock()
			}()
		}

		buf := getRandomDigits(num%concurrentRequests, true)
		numbers = append(numbers, buf...)
		buf = nil

		for ; ; {
			completedMutex.Lock()
			if completed == concurrentRequests {
				break
			}
			completedMutex.Unlock()
		}

	} else {
		numbers = getRandomDigits(num, true)
	}

	return numbers
}

func main() {
	var nums []int
	f, err := os.Open("nums.txt")

	// controlla se i numeri sono già stati scaricati e salvati
	if err == nil {
		// se si, i dati vengono caricati dal file
		buf, _ := ioutil.ReadAll(f)
		json.Unmarshal(buf, &nums)
		fmt.Println("Numbers loaded from file.")
	} else {
		// altrimenti scarica i numeri e li salva
		nums = getThreadedRandomDigits(total)
		buf, _ := json.Marshal(nums)
		ioutil.WriteFile("nums.txt", buf, 0644)
	}

	fmt.Println("Numbers loaded.")

	// prende i numeri in coppia e controlla se sono coprimi
	for x := 0; x < total-1; x += 2 {
		if gcd(nums[x], nums[x+1]) == 1 {
			coprimes++
		}

		if (x+2) % 1e4 == 0 {
			printProgress(x+2, total)
		}
	}

	// Usiamo totale/2 e non totale per i casi totali perchè consideriamo i numeri in coppia
	pi := math.Sqrt(6 * (total / 2) / float64(coprimes))
	fmt.Println(pi)
}
