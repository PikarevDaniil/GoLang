package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Safe struct {
	site  string
	login string
	pswd  string
}

type User struct {
	id   int64
	name string
	flag uint8
	box  Safe
}

func main() {
	the_bot()
}

func set_tools() (*sql.DB, *tg.BotAPI, tg.UpdatesChannel) {
	// mySQL connecting
	db, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/boxes")
	if err != nil {
		panic(err)
	}
	// Bot Settings
	bot, err := tg.NewBotAPI("token")
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Update Settings
	u := tg.NewUpdate(0)
	updates := bot.GetUpdatesChan(u)

	return db, bot, updates
}

func the_bot() {
	// Variables
	users := make(map[int64]User)
	db, bot, updates := set_tools()

	// Main loop
	for update := range updates {
		current_user, ok := users[update.FromChat().ID]
		if !ok {
			current_user = User{update.FromChat().ID, update.FromChat().UserName, 0, Safe{}}
		}

		if update.Message != nil {

			msg := tg.NewMessage(current_user.id, "") // Create a new Message instance
			log.Println("message " + update.Message.Text + " from " + current_user.name)
			request := "CREATE TABLE IF NOT EXISTS " + current_user.name + " (site varchar(255), login varchar(255), pswd varchar(255))"
			_, err := db.Exec(request)
			if err != nil {
				log.Println("creating error")
				log.Panic(err)
			}

			if !update.Message.IsCommand() {
				switch current_user.flag {
				case 0:
					msg.Text = "..."
				case 1, 2, 3:
					current_user, msg.Text = to_write(current_user, update.Message.Text, db)
				case 4:
					read_data(db, update.Message.Text, bot, current_user)
					current_user.flag = 0
					msg.Text = "Готово!"
				case 5:
					msg.Text = del_data(db, update.Message.Text, current_user)
					current_user.flag = 0
				}
			} else {

				// Handle the command
				switch update.Message.Command() {
				case "start":
					msg.Text = "Добро Пожаловать, " + update.Message.Chat.UserName
				case "help":
					msg.Text = "Этот бот хранит ваши данные в коробках"
				case "add":
					msg.Text = "Введите название сайта/софта..."
					current_user.flag = 1
				case "find":
					msg.Text = "Выберите сайт/софт..."
					msg.ReplyMarkup = build_kb(db, current_user)
					current_user.flag = 4
				case "del":
					msg.Text = "Введите название сайта/софта..."
					current_user.flag = 5
				default:
					msg.Text = "Я не знаю такой команды"
				}
			}

			// Update the user map with the new user_flag
			users[update.Message.Chat.ID] = current_user
			// Send the Message
			if _, err := bot.Send(msg); err != nil {
				log.Println("sending error")
				log.Panic(err)
			}
		} else if update.CallbackQuery != nil {

			callback := tg.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
			if _, err := bot.Request(callback); err != nil {
				panic(err)
			}

			read_data(db, update.CallbackQuery.Data, bot, current_user)
		}
	}
}

func to_write(user User, text string, db *sql.DB) (User, string) {
	out := "..."
	switch user.flag {
	case 1:
		user.box.site = text
		out = "Введите логин..."
		user.flag = 2
	case 2:
		user.box.login = text
		out = "Введите пароль..."
		user.flag = 3
	case 3:
		user.box.pswd = text
		out = "Готово"
		write_data(db, user)
		user.flag = 0
	}
	return user, out
}

func write_data(db *sql.DB, user User) {
	request := "INSERT INTO " + user.name + " (site, login, pswd) VALUES (?, ?, ?)"
	db.Exec(request, user.box.site, user.box.login, user.box.pswd)
}

func read_data(db *sql.DB, site string, bot *tg.BotAPI, user User) {
	out := "..."
	request := "SELECT login, pswd FROM " + user.name + " WHERE site =?"
	rows, err := db.Query(request, site)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("rows: %v", rows)
	defer rows.Close()
	var login string
	var pswd string
	for rows.Next() {
		err := rows.Scan(&login, &pswd)
		if err != nil {
			log.Panic(err)
		}
		out = "Логин: " + login + "\n Пароль: " + pswd
		msg := tg.NewMessage(user.id, out)
		bot.Send(msg)
	}
}

func del_data(db *sql.DB, site string, user User) string {
	request := "DELETE FROM " + user.name + " WHERE site =?"
	db.Exec(request, site)
	return "Готово!"
}

func build_kb(db *sql.DB, user User) tg.InlineKeyboardMarkup {
	var rows *sql.Rows
	var err error
	if rows, err = db.Query("SELECT DISTINCT site FROM " + user.name); err != nil {
		log.Panic(err)
	}
	defer rows.Close()

	var kb [][]tg.InlineKeyboardButton
	for rows.Next() {
		var site string
		err := rows.Scan(&site)
		if err != nil {
			log.Panic(err)
		}
		kb = append(kb, tg.NewInlineKeyboardRow(tg.NewInlineKeyboardButtonData(site, site)))
	}

	return tg.NewInlineKeyboardMarkup(kb...)
}
