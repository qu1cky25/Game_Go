package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type item struct {
	type_item string
	name      string
	attack    int
	defence   int
	plus_hp   int
}

type player struct {
	name      string
	hp        int
	inventory []item
	equipment []item
}

func get_safe_number(scanner *bufio.Scanner, prompt string, min, max int) int {
	for {
		fmt.Println(prompt)
		scanner.Scan()
		text := scanner.Text()
		if text == "exit" {
			return -1
		}
		num, err := strconv.Atoi(text)
		if err == nil && num >= min && num <= max {
			return num
		}
		fmt.Printf("Ошибка. Введите цифру от %d до %d\n", min, max)
	}
}

func main() {
	url := "http://localhost:8080"
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Введите ваше имя:")
	scanner.Scan()
	my_name := scanner.Text()
	http.Post(url, "text/plain", bytes.NewBufferString("NAME:"+my_name))

	me := &player{name: my_name, hp: 100}
	me.inventory = []item{
		{type_item: "оружие", name: "Вонючие носочки", attack: 30},
		{type_item: "броня", name: "Строительная каска", defence: 50},
		{type_item: "хилка", name: "Пицца", plus_hp: 20},
	}

	last_log_len := 0
	for {
		resp, _ := http.Get(url)
		body, _ := io.ReadAll(resp.Body)
		logs := strings.Split(strings.TrimSpace(string(body)), "\n")

		if len(logs) > last_log_len {
			for i := last_log_len; i < len(logs); i++ {
				fmt.Println(logs[i])
				if strings.Contains(logs[i], "Ожидание хода противника.") {

					move_done := false
					for !move_done {
						fmt.Println("\n1. Атаковать\n2. Экипировать\n3. Инвентарь\n4. Снять")
						choice := get_safe_number(scanner, "Ваш выбор:", 1, 4)
						if choice == -1 {
							http.Post(url, "text/plain", bytes.NewBufferString("exit"))
							os.Exit(0)
						}

						switch choice {
						case 1:
							h := get_safe_number(scanner, "Атака (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие):", 1, 5)
							b := get_safe_number(scanner, "Защита (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие):", 1, 5)

							points := map[int]string{1: "уши", 2: "глаза", 3: "нос", 4: "правое полушарие", 5: "левое полушарие"}
							move_str := points[h] + ":" + points[b]
							http.Post(url, "text/plain", bytes.NewBufferString(move_str))
							move_done = true
						case 2:
							for i, it := range me.inventory {
								fmt.Printf("%d. %s\n", i+1, it.name)
							}
							idx := get_safe_number(scanner, "Индекс:", 1, len(me.inventory)) - 1
							item := me.inventory[idx]
							if item.type_item == "хилка" {
								me.hp += item.plus_hp
								me.inventory = append(me.inventory[:idx], me.inventory[idx+1:]...)
							} else {
								me.equipment = append(me.equipment, item)
								me.inventory = append(me.inventory[:idx], me.inventory[idx+1:]...)
							}
							fmt.Println("Предмет надет.")
						case 3:
							fmt.Printf("HP: %d, инвентарь: %v\n", me.hp, me.inventory)
						case 4:
							for i, it := range me.equipment {
								fmt.Printf("%d. %s\n", i+1, it.name)
							}
							idx := get_safe_number(scanner, "Индекс:", 1, len(me.equipment)) - 1
							me.inventory = append(me.inventory, me.equipment[idx])
							me.equipment = append(me.equipment[:idx], me.equipment[idx+1:]...)
							fmt.Println("Предмет снят.")
						}
					}
				}
			}
			last_log_len = len(logs)
		}
		if strings.Contains(string(body), "ИГРА ОКОНЧЕНА") {
			break
		}
		time.Sleep(1 * time.Second)
	}
}
