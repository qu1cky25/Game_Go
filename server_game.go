package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Character interface {
	hit_target(target Character, hit_point string)
	block_attack(hit_point string) bool
	get_hp() int
	get_name() string
}

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
	strength  int
	hit       string
	block     string
	inventory []item
	equipment []item
}

func (p *player) get_hp() int {
	return p.hp
}
func (p *player) get_name() string {
	return p.name
}

func (p *player) hit_target(target Character, hit_point string) {
	damage := p.strength
	for _, it := range p.equipment {
		if it.type_item == "оружие" {
			damage += it.attack
		}
	}

	switch hit_point {
	case "уши":
		damage += 5
	case "глаза":
		damage += 10
	case "нос":
		damage += 5
	case "правое полушарие":
		damage += 15
	case "левое полушарие":
		damage += 20
	}

	if !target.block_attack(hit_point) {
		t := target.(*player)
		has_armor := false
		for i := 0; i < len(t.equipment); i++ {
			if t.equipment[i].type_item == "броня" {
				has_armor = true
				t.equipment[i].defence -= damage
				add_game_log(fmt.Sprintf("Броня игрока %s поглотила урон (осталось прочности: %d).", t.name, t.equipment[i].defence))

				if t.equipment[i].defence <= 0 {
					add_game_log(fmt.Sprintf("Броня игрока %s сломалась.", t.name))
					t.equipment = append(t.equipment[:i], t.equipment[i+1:]...)
				}
				break
			}
		}

		if !has_armor {
			t.hp -= damage
			add_game_log(fmt.Sprintf("Игрок %s получил %d урона в %s.", t.name, damage, hit_point))
		}
	} else {
		add_game_log(fmt.Sprintf("Игрок %s заблокировал удар в %s.", target.get_name(), hit_point))
	}
}

func (p *player) block_attack(hit_point string) bool {
	return p.block == hit_point
}

var (
	game_history       []string
	history_mu         sync.Mutex
	client_action_chan = make(chan string, 1)
	p2_name            string
	game_running       = true
)

func add_game_log(msg string) {
	history_mu.Lock()
	game_history = append(game_history, msg)
	fmt.Println(msg)
	history_mu.Unlock()
}

func get_point_name(idx int) string {
	points := map[int]string{1: "уши", 2: "глаза", 3: "нос", 4: "правое полушарие", 5: "левое полушарие"}
	return points[idx]
}

func get_safe_number(scanner *bufio.Scanner, mess string, min, max int) int {
	for {
		fmt.Println(mess)
		scanner.Scan()
		text := scanner.Text()
		if text == "exit" {
			return -1
		}
		num, err := strconv.Atoi(text)
		if err == nil && num >= min && num <= max {
			return num
		}
		fmt.Printf("Ошибка. Введите число от %d до %d или 'exit'\n", min, max)
	}
}

func manage_inventory(p *player, scanner *bufio.Scanner) {
	for {
		fmt.Println("\n1. Экипировать\n2. Снять\n3. Показать инвентарь\n4. Назад")
		choice := get_safe_number(scanner, "Выберите действие:", 1, 4)
		if choice == 4 || choice == -1 {
			return
		}

		switch choice {
		case 1:
			if len(p.inventory) == 0 {
				fmt.Println("Инвентарь пуст.")
				continue
			}
			for i, it := range p.inventory {
				fmt.Printf("%d. %s (%s)\n", i+1, it.name, it.type_item)
			}
			idx := get_safe_number(scanner, "Что надеть/взять?", 1, len(p.inventory)) - 1
			item := p.inventory[idx]

			if item.type_item == "хилка" {
				p.hp += item.plus_hp
				add_game_log(p.name + " применил хилку и восстановил здоровье.")
				p.inventory = append(p.inventory[:idx], p.inventory[idx+1:]...)
			} else {
				p.equipment = append(p.equipment, item)
				p.inventory = append(p.inventory[:idx], p.inventory[idx+1:]...)
				add_game_log(p.name + " экипировал " + item.name)
			}
		case 2:
			if len(p.equipment) == 0 {
				fmt.Println("Ничего не надето.")
				continue
			}
			for i, it := range p.equipment {
				fmt.Printf("%d. %s\n", i+1, it.name)
			}
			idx := get_safe_number(scanner, "Что снять?", 1, len(p.equipment)) - 1
			p.inventory = append(p.inventory, p.equipment[idx])
			add_game_log(p.name + " снял " + p.equipment[idx].name)
			p.equipment = append(p.equipment[:idx], p.equipment[idx+1:]...)
		case 3:
			fmt.Println("Здоровье:", p.hp)
			fmt.Println("Инвентарь:", p.inventory)
			fmt.Println("Экипировка:", p.equipment)
		}
	}
}

func server_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		history_mu.Lock()
		fmt.Fprint(w, strings.Join(game_history, "\n"))
		history_mu.Unlock()
	} else if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		msg := string(body)
		if strings.HasPrefix(msg, "NAME:") {
			p2_name = strings.TrimPrefix(msg, "NAME:")
			add_game_log("Игрок " + p2_name + " подключился!")
		} else if msg == "exit" {
			add_game_log("Противник покинул игру.")
			game_running = false
		} else {
			client_action_chan <- msg
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	http.HandleFunc("/", server_handler)
	go http.ListenAndServe(":8080", nil)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Введите имя сервера:")
	scanner.Scan()
	p1 := &player{name: scanner.Text(), hp: 100, strength: 10}

	p1.inventory = append(p1.inventory, item{type_item: "оружие", name: "Вонючие носочки", attack: 30})
	p1.inventory = append(p1.inventory, item{type_item: "броня", name: "Строительная каска", defence: 50})
	p1.inventory = append(p1.inventory, item{type_item: "хилка", name: "Пицца", plus_hp: 20})

	fmt.Println("Ожидание клиента...")
	for p2_name == "" {
		time.Sleep(1 * time.Second)
	}
	p2 := &player{name: p2_name, hp: 100, strength: 10}
	p2.inventory = p1.inventory

	for p1.hp > 0 && p2.hp > 0 && game_running {
		add_game_log(fmt.Sprintf("\n----- ХОД ИГРОКА %s -----", p1.name))
		move_done := false
		for !move_done {
			fmt.Println("1. Атаковать\n2. Экипировать\n3. Показать инвентарь\n4. Снять предмет")
			choice := get_safe_number(scanner, "Ваш выбор:", 1, 4)
			if choice == -1 {
				game_running = false
				break
			}

			switch choice {
			case 1:
				hit := get_safe_number(scanner, "Куда бьем? (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие):", 1, 5)
				block := get_safe_number(scanner, "Что защищаем? (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие):", 1, 5)
				p1.hit, p1.block = get_point_name(hit), get_point_name(block)
				move_done = true
			case 2, 3, 4:
				manage_inventory(p1, scanner)
			}
		}

		if !game_running {
			break
		}
		add_game_log("Ожидание хода противника...")
		client_move := strings.Split(<-client_action_chan, ":")
		p2.hit, p2.block = client_move[0], client_move[1]

		add_game_log(p1.name + " атаковал.")
		add_game_log(p2.name + " атаковал.")

		p1.hit_target(p2, p1.hit)
		p2.hit_target(p1, p2.hit)
	}
	add_game_log("ИГРА ОКОНЧЕНА")
}
