package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func set_tools() (*sql.DB, *tgbotapi.BotAPI, tgbotapi.UpdatesChannel) {
	// mySQL connecting
	db, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/boxes")
	if err != nil {
		panic(err)
	}
	// Bot Settings
	bot, err := tgbotapi.NewBotAPI("telegram_token")
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Update Settings
	u := tgbotapi.NewUpdate(0)
	updates := bot.GetUpdatesChan(u)

	return db, bot, updates
}

func the_bot() {
	// Variables
	users := make(map[int64]User)
	db, bot, updates := set_tools()

	// Main loop
	for update := range updates {
		current_user, ok := users[update.Message.Chat.ID]
		if !ok {
			current_user = User{update.Message.Chat.ID, update.Message.Chat.UserName, 0, Safe{}}
		}

		msg := tgbotapi.NewMessage(current_user.id, "") // Create a new Message instance
		log.Println(current_user.name + " is here!")
		request := "CREATE TABLE IF NOT EXISTS " + current_user.name + " (site varchar(255), login varchar(255), pswd varchar(255))"
		_, err := db.Exec(request)
		if err != nil {
			log.Panic(err)
		}

		// Ignore any non-Message Updates
		if update.Message == nil {
			continue
		}

		// anser to non-command message
		if !update.Message.IsCommand() {
			switch current_user.flag {
			case 0:
				msg.Text = "..."
			case 1, 2, 3:
				current_user, msg.Text = to_write(current_user, update.Message.Text, db)
			case 4:
				read_data(db, update.Message.Text, bot, current_user)
				msg.Text = "Done!"
			case 5:
				msg.Text = del_data(db, update.Message.Text, current_user)
			}
		} else {

			// Handle the command
			switch update.Message.Command() {
			case "start":
				msg.Text = "Hi, " + update.Message.Chat.UserName
			case "help":
				msg.Text = "I have /add, /find, /del commands for working with your data"
			case "add":
				msg.Text = "Enter site/software name..."
				current_user.flag = 1
			case "find":
				msg.Text = "Enter site/software name..."
				current_user.flag = 4
			case "del":
				msg.Text = "Enter site/software name..."
				current_user.flag = 5
			default:
				msg.Text = "I don't know that command"
			}
		}

		// Update the user map with the new user_flag
		users[update.Message.Chat.ID] = current_user
		// Send the Message
		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

func to_write(user User, text string, db *sql.DB) (User, string) {
	out := "..."
	switch user.flag {
	case 1:
		user.box.site = text
		out = "Enter login..."
		user.flag = 2
	case 2:
		user.box.login = text
		out = "Enter password..."
		user.flag = 3
	case 3:
		user.box.pswd = text
		out = "Done"
		write_data(db, user)
		log.Println(user.box)
		user.flag = 0
	}
	return user, out
}

func write_data(db *sql.DB, user User) {
	request := "INSERT INTO " + user.name + " (site, login, pswd) VALUES (?, ?, ?)"
	db.Exec(request, user.box.site, user.box.login, user.box.pswd)
}

func read_data(db *sql.DB, site string, bot *tgbotapi.BotAPI, user User) {
	out := "..."
	request := "SELECT login, pswd FROM " + user.name + " WHERE site =?"
	rows, err := db.Query(request, site)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	var login string
	var pswd string
	for rows.Next() {
		err := rows.Scan(&login, &pswd)
		if err != nil {
			log.Panic(err)
		}
		out = "Login: " + login + "\nPassword: " + pswd
		msg := tgbotapi.NewMessage(user.id, out)
		bot.Send(msg)
	}
}

func del_data(db *sql.DB, site string, user User) string {
	request := "DELETE FROM " + user.name + " WHERE site =?"
	db.Exec(request, site)
	return "Done!"
}
