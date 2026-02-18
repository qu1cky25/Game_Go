package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type player struct {
	name string
	hp   int
}

func play_сlient() {
	url := ""

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Введите ваше имя: ")
	scanner.Scan()
	my_name := scanner.Text()

	http.Post(url, "text/plain", bytes.NewBufferString("JOIN:"+my_name))

	last_log_size := 0
	for {
		resp, err := http.Get(url)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			lines := strings.Split(strings.TrimSpace(string(body)), "\n")

			if len(lines) > last_log_size && lines[0] != "" {
				for i := last_log_size; i < len(lines); i++ {
					line := lines[i]
					fmt.Println(line)
					if strings.Contains(line, "Ожидаем ход игрока "+my_name) {
						fmt.Println("\nВаш ход (введите две цифры через пробел: 1-я цифра - атака, 2-я - зашита).")
						fmt.Println("1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие")
						scanner.Scan()
						move := scanner.Text()
						http.Post(url, "text/plain", bytes.NewBufferString(move))
					}
				}
				last_log_size = len(lines)
			}
			resp.Body.Close()

			if strings.Contains(string(body), "ИГРА ОКОНЧЕНА") {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	fmt.Println("1. Подключиться к сетевому PvP")
	var choice int
	fmt.Scanln(&choice)
	if choice == 1 {
		play_сlient()
	}
}
