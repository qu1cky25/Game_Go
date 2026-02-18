package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
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

func (p *player) get_hp() int      { return p.hp }
func (p *player) get_name() string { return p.name }

func (p *player) block_attack(hit_point string) bool {
	return p.block == hit_point
}

func get_damage_by_point(point string) int {
	switch point {
	case "уши":
		return 5
	case "глаза":
		return 10
	case "нос":
		return 5
	case "правое полушарие":
		return 15
	case "левое полушарие":
		return 20
	default:
		return 0
	}
}

func get_point_name(index string) string {
	switch index {
	case "1":
		return "уши"
	case "2":
		return "глаза"
	case "3":
		return "нос"
	case "4":
		return "правое полушарие"
	case "5":
		return "левое полушарие"
	default:
		return "уши"
	}
}

func (p *player) hit_target(target Character, hit_point string) {
	damage := get_damage_by_point(hit_point)
	if !target.block_attack(hit_point) {
		if t, ok := target.(*player); ok {
			t.hp -= damage
			add_log(fmt.Sprintf("[УДАР] %s попал в %s. Нанесено %d урона.", p.name, hit_point, damage))
		}
	} else {
		add_log(fmt.Sprintf("[БЛОК] %s защитил %s.", target.get_name(), hit_point))
	}
}

var (
	game_log     []string
	log_mutex    sync.Mutex
	client_move  = make(chan string, 1)
	client_name  string
	start_signal = make(chan bool, 1)
)

func add_log(msg string) {
	log_mutex.Lock()
	game_log = append(game_log, msg)
	fmt.Println(msg)
	log_mutex.Unlock()
}

func server_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		log_mutex.Lock()
		fmt.Fprint(w, strings.Join(game_log, "\n"))
		log_mutex.Unlock()
	} else if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		data := string(body)
		if strings.HasPrefix(data, "JOIN:") {
			client_name = strings.TrimPrefix(data, "JOIN:")
			start_signal <- true
		} else {
			client_move <- data
		}
		fmt.Fprint(w, "OK")
	}
}

func start_battle_server() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Введите ваше имя: ")
	scanner.Scan()
	srv_name := scanner.Text()

	p1 := &player{name: srv_name, hp: 100}

	add_log("Ожидание подключения игрока...")
	<-start_signal
	p2 := &player{name: client_name, hp: 100}
	add_log("Игрок " + p2.name + " подключился.")

	for p1.hp > 0 && p2.hp > 0 {
		add_log(fmt.Sprintf("\n--- Статус: %s [%d HP] | %s [%d HP] ---", p1.name, p1.hp, p2.name, p2.hp))
		add_log("Выберите действие (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие)")

		fmt.Print("Ваш ход (введите две цифры через пробел: 1-я цифра - атака, 2-я - зашита): ")
		scanner.Scan()
		input := strings.Split(scanner.Text(), " ")
		p1.hit = get_point_name(input[0])
		p1.block = get_point_name(input[1])

		add_log("Ожидаем ход игрока " + p2.name)
		cInput := strings.Split(<-client_move, " ")
		p2.hit = get_point_name(cInput[0])
		p2.block = get_point_name(cInput[1])
		p1.hit_target(p2, p1.hit)
		p2.hit_target(p1, p2.hit)
	}

	if p1.hp <= 0 && p2.hp <= 0 {
		add_log("НИЧЬЯ")
	} else if p1.hp <= 0 {
		add_log("ПОБЕДИТЕЛЬ: " + p2.name)
	} else {
		add_log("ПОБЕДИТЕЛЬ: " + p1.name)
	}
	add_log("ИГРА ОКОНЧЕНА")
}

func main() {
	http.HandleFunc("/", server_handler)
	go http.ListenAndServe(":8080", nil)

	fmt.Println("1. Сюжет\n2. PvP сетевой")
	var choice int
	fmt.Scanln(&choice)
	if choice == 2 {
		start_battle_server()
	} else {
		fmt.Println("Запустите оригинальный файл для сюжета.")
	}
}
