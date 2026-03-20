package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// структуры и интерфейсы
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

type enemy struct {
	name     string
	hp       int
	strength int
	hit      string
	block    string
	trophy   *item
}

func (p *player) get_hp() int                        { return p.hp }
func (p *player) get_name() string                   { return p.name }
func (e *enemy) get_hp() int                         { return e.hp }
func (e *enemy) get_name() string                    { return e.name }
func (p *player) block_attack(hit_point string) bool { return p.block == hit_point }
func (e *enemy) block_attack(hit_point string) bool  { return e.block == hit_point }

func (p *player) hit_target(target Character, hit_point string) {
	damage := p.strength
	weapon_idx := -1

	for i, it := range p.equipment {
		if it.type_item == "оружие" {
			damage += it.attack
			weapon_idx = i
			break
		}
	}

	switch hit_point {
	case "уши", "нос":
		damage += 5
	case "глаза":
		damage += 10
	case "правое полушарие":
		damage += 15
	case "левое полушарие":
		damage += 20
	}

	if weapon_idx != -1 {
		fmt.Printf("Игрок %s использовал оружие '%s' (+%d урона), и оно сломалось.\n", p.name, p.equipment[weapon_idx].name, p.equipment[weapon_idx].attack)
		p.equipment = append(p.equipment[:weapon_idx], p.equipment[weapon_idx+1:]...)
	}

	if !target.block_attack(hit_point) {
		has_armor := false
		if t, ok := target.(*player); ok {
			for i := 0; i < len(t.equipment); i++ {
				if t.equipment[i].type_item == "броня" {
					has_armor = true
					t.equipment[i].defence -= damage
					if t.equipment[i].defence <= 0 {
						fmt.Printf("Броня игрока %s полностью сломалась!\n", t.name)
						t.equipment = append(t.equipment[:i], t.equipment[i+1:]...)
					} else {
						fmt.Printf("Броня игрока %s поглотила урон (осталось прочности: %d).\n", t.name, t.equipment[i].defence)
					}
					break
				}
			}
			if !has_armor {
				t.hp -= damage
				fmt.Printf("Игрок %s получил %d урона в %s.\n", t.name, damage, hit_point)
			}
		} else if t, ok := target.(*enemy); ok {
			t.hp -= damage
			fmt.Printf("Противник %s получил %d урона в %s.\n", t.name, damage, hit_point)
		}
	} else {
		fmt.Printf("Удар в %s успешно заблокирован!\n", hit_point)
	}
}

func (e *enemy) hit_target(target Character, hit_point string) {
	damage := e.strength
	switch hit_point {
	case "уши", "нос":
		damage += 5
	case "глаза":
		damage += 10
	case "правое полушарие":
		damage += 15
	case "левое полушарие":
		damage += 20
	}
	if !target.block_attack(hit_point) {
		if t, ok := target.(*player); ok {
			has_armor := false
			for i := 0; i < len(t.equipment); i++ {
				if t.equipment[i].type_item == "броня" {
					has_armor = true
					t.equipment[i].defence -= damage
					if t.equipment[i].defence <= 0 {
						fmt.Printf("Броня игрока %s полностью сломалась!\n", t.name)
						t.equipment = append(t.equipment[:i], t.equipment[i+1:]...)
					} else {
						fmt.Printf("Броня игрока %s поглотила урон от врага (осталось прочности: %d).\n", t.name, t.equipment[i].defence)
					}
					break
				}
			}
			if !has_armor {
				t.hp -= damage
				fmt.Printf("Противник %s нанес вам %d урона в %s!\n", e.name, damage, hit_point)
			}
		}
	} else {
		fmt.Printf("Вы успешно заблокировали удар врага в %s.\n", hit_point)
	}
}

// вспомогательные функции
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

func get_valid_name(scanner *bufio.Scanner, prompt string) string {
	for {
		fmt.Println(prompt)
		scanner.Scan()
		name := strings.TrimSpace(scanner.Text())
		if name != "" && name != "exit" {
			return name
		}
		fmt.Println("Имя не может быть пустым или 'exit'.")
	}
}

func show_inventory(inv []item) {
	fmt.Println("I. Оружие")
	for i, it := range inv {
		if it.type_item == "оружие" {
			fmt.Printf("\t%d. %s (%d ед. урона)\n", i+1, it.name, it.attack)
		}
	}
	fmt.Println("II. Броня")
	for i, it := range inv {
		if it.type_item == "броня" {
			fmt.Printf("\t%d. %s (%d ед. прочности)\n", i+1, it.name, it.defence)
		}
	}
	fmt.Println("III. Хилки")
	for i, it := range inv {
		if it.type_item == "хилка" {
			fmt.Printf("\t%d. %s (восстанавливает %d хп)\n", i+1, it.name, it.plus_hp)
		}
	}
}

func show_and_choose_inventory(scanner *bufio.Scanner, inv []item, action string) int {
	for {
		show_inventory(inv)
		fmt.Printf("Введите номер предмета чтобы %s (или 'exit' для отмены): \n", action)
		scanner.Scan()
		text := scanner.Text()
		if text == "exit" {
			return -1
		}
		num, err := strconv.Atoi(text)
		if err == nil && num >= 1 && num <= len(inv) {
			return num - 1
		}
		fmt.Println("Ошибка! Неверный номер предмета.")
	}
}

func get_random_trophy() *item {
	trophies := []item{
		{"оружие", "Крик", 15, 0, 0},
		{"оружие", "Фразы", 10, 0, 0},
		{"оружие", "Указка", 12, 0, 0},
		{"оружие", "Вонючие носочки", 30, 0, 0},
		{"броня", "Кепка", 0, 10, 0},
		{"броня", "Бронежилет охранника", 0, 60, 0},
		{"броня", "Строительная каска", 0, 50, 0},
		{"хилка", "Студенческий хот-дог", 0, 0, 30},
		{"хилка", "Пицца", 0, 0, 20},
		{"хилка", "Сэндвич", 0, 0, 15},
	}
	selected := trophies[rand.Intn(len(trophies))]
	return &selected
}

func local_player_menu(scanner *bufio.Scanner, p *player) bool {
	move_done := false
	for !move_done {
		fmt.Printf("\n--- ХОД ИГРОКА %s (%d HP) ---\n", p.name, p.hp)
		fmt.Println("1. Атаковать\n2. Экипировать\n3. Показать инвентарь\n4. Снять предмет")
		choice := get_safe_number(scanner, "Ваш выбор:", 1, 4)
		if choice == -1 {
			return false
		}

		switch choice {
		case 1:
			hit := get_safe_number(scanner, "Куда бьем? (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие):", 1, 5)
			block := get_safe_number(scanner, "Что защищаем? (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие):", 1, 5)
			p.hit, p.block = get_point_name(hit), get_point_name(block)
			move_done = true
		case 2:
			if len(p.inventory) == 0 {
				fmt.Println("Инвентарь пуст.")
				continue
			}
			idx := show_and_choose_inventory(scanner, p.inventory, "экипировать")
			if idx != -1 {
				it := p.inventory[idx]
				if it.type_item == "хилка" {
					p.hp += it.plus_hp
					fmt.Printf("Вы применили '%s' и восстановили %d хп.\n", it.name, it.plus_hp)
				} else {
					p.equipment = append(p.equipment, it)
					fmt.Printf("Вы экипировали '%s'.\n", it.name)
				}
				p.inventory = append(p.inventory[:idx], p.inventory[idx+1:]...)
			}
		case 3:
			fmt.Printf("HP: %d\n--- Инвентарь ---\n", p.hp)
			show_inventory(p.inventory)
			fmt.Println("--- Экипировано ---")
			show_inventory(p.equipment)
		case 4:
			if len(p.equipment) == 0 {
				fmt.Println("Ничего не надето.")
				continue
			}
			idx := show_and_choose_inventory(scanner, p.equipment, "снять")
			if idx != -1 {
				it := p.equipment[idx]
				p.inventory = append(p.inventory, it)
				p.equipment = append(p.equipment[:idx], p.equipment[idx+1:]...)
				fmt.Printf("Предмет '%s' снят.\n", it.name)
			}
		}
	}
	return true
}

// локальные режимы
type scenario struct {
	day_text   string
	enemy_name string
	enemy_hp   int
}

func play_story(scanner *bufio.Scanner) {
	fmt.Println("Нашего главного героя зовут Денис. Он студент 3-го курса КТК...")
	p := &player{name: "Денис", hp: 200, strength: 10, inventory: []item{
		{type_item: "оружие", name: "Ручка", attack: 10},
		{type_item: "хилка", name: "Сухарики", plus_hp: 10},
	}}

	days := []scenario{
		{
			day_text:   
			"Вступление.\nДенис наконец-то закончил очередной курс колледжа и собирался приступить к новому этапу учебы.\n
			Проснувшись рано утром, он быстро собрался и направился в сторону учебного корпуса.\n
			Надежд было полно, стремление учится находилось на высоте! Но, кое что с ним произошло...\n
			Первый день.\nУтро началось спокойно. Денис проснулся, как обычно нерасторопно собирался в колледж.\n
			Ничего не предвещало беды, первый день всё таки, но подойдя к первому перекрёстку.\n
			Здесь, словно назло, стоит молодой парень, известный всей округе своей дерзостью\n 
			и любовью к приключениям. Звать его Рома.
			",
			enemy_name: "Гопник Рома", enemy_hp: 100,
		},
		{
			day_text:   "Второй день.\n
			Наступил второй день. Денис только сейчас смог отойти, после вчерашнего инцидента.\n
			В этот раз он решил пойти по другому маршруту, в надежде на то, что сейчас, всё\n
			будет иначе и ему удастся спокойно дойти. ройдясь по улочкам, его ждал новый соперник,\n
			менее агрессивный внешне, но куда более непредсказуемый внутри. Зверь всей сети магазинов КБ...\n
			И имя его Колян, бывший триллионер из трущоб, ныне генерал 'пустая-бутылка'.\n
			Он повернулся резко, как хищник и завыл:\n
			- Я... это уб$!@%#ф. Ик! Сто-о-о-о-Ять! Теб-бь-бь-бь-бь-бяяяяяя ща (#!@@$%**!\n
			Его руки дрожат, как отбойный молоток, глаза словно помидоры, но дух ещё силён.\n
			Несмотря на слабость тела, Колян умело владеет приёмами уличных боевых искусств.\n
			Если верить его словам... Колледж совсем близко, но продвижение остановлено\n
			новым ТЯЖЕЛЕИШИМ испытанием.
			",
			enemy_name: "алкаш Колян aka 'Пустая бутылка'", enemy_hp: 150,
		},
		{
			day_text:   "Третий день.\nТретий день был для Дениса туманным. Он же не знал, что можно ожидать,\n
			то гопники, а вчера вообще пьяница напал! В этот раз наш студент,\n
			морально готовится к худшем. Долго идя до места назначения, он никого не повстречал\n
			и вдруг его посетила мысль: 'Может сегодня, всё будет хорошо? Без всяких приключений!',
			но как бы не так... Перед колледжем возвышалась фигура мрачного лидера местных обитателей улиц заставляя\n
			- князь всея бомжей Василий. Этот хитрый старик стал настоящим хозяином территории,\n
			всех подчиняться своим правилам. Одинокий, грозный и очень вонючий, великий князь заговорил:\n
			-Эй, малой! А ну ка, быстро накидал мне мелочи, для моего целебного элексира иначе... Я просто выбью их силой!\n
			Последний бой приближается. За воротами колледжа ждут пары, но перед ними преграда - таинственный правитель\n
			нищих, способный поставить крест на учёбе.
			",
			enemy_name: "Король бомжиков Василий", enemy_hp: 200,
		},
	}

	for i, scen := range days {
		fmt.Printf("\n=== %s ===\n", scen.day_text)
		e := &enemy{name: scen.enemy_name, hp: scen.enemy_hp, strength: 10}
		if i < 2 {
			e.trophy = get_random_trophy()
		}

		for p.hp > 0 && e.hp > 0 {
			if !local_player_menu(scanner, p) {
				return
			}

			e.hit = get_point_name(rand.Intn(5) + 1)
			e.block = get_point_name(rand.Intn(5) + 1)

			p.hit_target(e, p.hit)
			e.hit_target(p, e.hit)
			fmt.Printf("=== Итоги раунда: Здоровье %s: %d HP | Здоровье %s: %d HP ===\n", p.name, p.hp, e.name, e.hp)
		}

		if p.hp <= 0 {
			fmt.Printf("К сожалению, вы проиграли... Игра окончена.\n")
			return
		} else {
			fmt.Printf("Вы победили %s!\n", e.name)
			if e.trophy != nil {
				fmt.Printf("Трофей: вы получили '%s'!\n", e.trophy.name)
				p.inventory = append(p.inventory, *e.trophy)
			}
		}	
	}
	fmt.Println("Концовка.\n
	После трёх адских дней, всё утихло. Денис смог показать всем, что он не просто сопляк,\n
	а настоящий студент КТК! И его никто не остановит, перед стремление к учёбе. После данных инцидентов,\n
	к нему перестали приставать, его боялись, а кто то даже уважал. князь Василий, ушёл в забвение.\n
	Теперь Денис может спокойно ходить в колледж. Ему больше ничего не угрожает. Хаппа энда. 
	") //хэппи энд
}

func play_hotseat(scanner *bufio.Scanner) {
	fmt.Println("Введите имя Игрока 1:")
	p1_name := get_valid_name(scanner, "")
	fmt.Println("Введите имя Игрока 2:")
	p2_name := get_valid_name(scanner, "")

	p1 := &player{name: p1_name, hp: 100, strength: 10, inventory: []item{
		{type_item: "оружие", name: "Карандаш", attack: 10},
		{type_item: "хилка", name: "Чай", plus_hp: 15},
	}}
	p2 := &player{name: p2_name, hp: 100, strength: 10, inventory: []item{
		{type_item: "оружие", name: "Линейка", attack: 12},
		{type_item: "хилка", name: "Кофе", plus_hp: 15},
	}}

	for p1.hp > 0 && p2.hp > 0 {
		if !local_player_menu(scanner, p1) {
			return
		}
		fmt.Println("\n\n\n\n[Передайте клавиатуру второму игроку]")
		if !local_player_menu(scanner, p2) {
			return
		}

		p1.hit_target(p2, p1.hit)
		p2.hit_target(p1, p2.hit)
		fmt.Printf("=== Итоги раунда: Здоровье %s: %d HP | Здоровье %s: %d HP ===\n", p1.name, p1.hp, p2.name, p2.hp)
	}

	if p1.hp <= 0 && p2.hp <= 0 {
		fmt.Println("Бой окончился ничьей!")
	} else if p1.hp <= 0 {
		fmt.Printf("Победил игрок %s!\n", p2.name)
	} else if p2.hp <= 0 {
		fmt.Printf("Победил игрок %s!\n", p1.name)
	}
}

// сетевой режим
func play_network_client(scanner *bufio.Scanner) {
	fmt.Println("Введите URL сервера:")
	scanner.Scan()
	url := strings.TrimSpace(scanner.Text())

	fmt.Println("Введите ваше имя:")
	my_name := get_valid_name(scanner, "")
	http.Post(url, "text/plain", bytes.NewBufferString("NAME:"+my_name))

	me := &player{name: my_name, hp: 100, strength: 10}
	me.inventory = []item{
		{type_item: "оружие", name: "Вонючие носочки", attack: 30},
		{type_item: "оружие", name: "Фонарик на телефоне", attack: 20},
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
				if strings.Contains(logs[i], "Ожидание хода противника...") {

					move_done := false
					for !move_done {
						fmt.Println("\n1. Атаковать\n2. Экипировать\n3. Показать инвентарь\n4. Снять предмет\n5. Написать сообщение")
						choice := get_safe_number(scanner, "Ваш выбор:", 1, 5)
						if choice == -1 {
							http.Post(url, "text/plain", bytes.NewBufferString("exit"))
							os.Exit(0)
						}

						switch choice {
						case 1:
							h := get_safe_number(scanner, "Куда бьем? (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие):", 1, 5)
							b := get_safe_number(scanner, "Что защищаем? (1 - уши, 2 - глаза, 3 - нос, 4 - правое полушарие, 5 - левое полушарие):", 1, 5)
							points := map[int]string{1: "уши", 2: "глаза", 3: "нос", 4: "правое полушарие", 5: "левое полушарие"}
							move_str := points[h] + ":" + points[b]

							for idx, it := range me.equipment {
								if it.type_item == "оружие" {
									me.equipment = append(me.equipment[:idx], me.equipment[idx+1:]...)
									break
								}
							}
							http.Post(url, "text/plain", bytes.NewBufferString(move_str))
							move_done = true
						case 2:
							if len(me.inventory) == 0 {
								fmt.Println("Инвентарь пуст.")
								continue
							}
							idx := show_and_choose_inventory(scanner, me.inventory, "экипировать")
							if idx != -1 {
								item := me.inventory[idx]
								if item.type_item == "хилка" {
									me.hp += item.plus_hp
									http.Post(url, "text/plain", bytes.NewBufferString(fmt.Sprintf("HEAL:%d", item.plus_hp)))
								} else {
									me.equipment = append(me.equipment, item)
									val := item.attack
									if item.type_item == "броня" {
										val = item.defence
									}
									http.Post(url, "text/plain", bytes.NewBufferString(fmt.Sprintf("EQUIP:%s:%s:%d", item.type_item, item.name, val)))
								}
								me.inventory = append(me.inventory[:idx], me.inventory[idx+1:]...)
								fmt.Println("Действие выполнено.")
							}
						case 3:
							fmt.Printf("HP: %d\n--- Инвентарь ---\n", me.hp)
							show_inventory(me.inventory)
							fmt.Println("--- Экипировано ---")
							show_inventory(me.equipment)
						case 4:
							if len(me.equipment) == 0 {
								fmt.Println("Ничего не надето.")
								continue
							}
							idx := show_and_choose_inventory(scanner, me.equipment, "снять")
							if idx != -1 {
								item := me.equipment[idx]
								me.inventory = append(me.inventory, item)
								me.equipment = append(me.equipment[:idx], me.equipment[idx+1:]...)
								http.Post(url, "text/plain", bytes.NewBufferString("UNEQUIP:"+item.name))
								fmt.Println("Предмет снят.")
							}
						case 5:
							fmt.Println("Введите сообщение:")
							scanner.Scan()
							msg := scanner.Text()
							http.Post(url, "text/plain", bytes.NewBufferString("CHAT:"+msg))
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

// главная функция
func main() {
	rand.Seed(time.Now().UnixNano())
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("\n----- МЕНЮ -----")
		fmt.Println("1. Играть в одиночную игру (сюжет)")
		fmt.Println("2. Играть в PvP за одним компьютером")
		fmt.Println("3. Играть в сетевой PvP")

		choice := get_safe_number(scanner, "Ваш выбор:", 1, 3)

		if choice == 1 {
			play_story(scanner)
		} else if choice == 2 {
			play_hotseat(scanner)
		} else if choice == 3 {
			play_network_client(scanner)
			break
		}
	}
}
