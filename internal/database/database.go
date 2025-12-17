package database

import (
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDB() {
	const (
                dbUser     = "ikasrfuasfhajh"
                dbPassword = "kH8lK8dD8qkH8lK8dD8qwqeaxczxc"
                dbHost     = "wh26866.web4.maze-tech.ru"
                dbName     = "newsite"
	)

        dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbName)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {	
		log.Fatalf("Ошибка при открытии соединения с БД: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Ошибка при проверке соединения с БД: %v", err)
	}

	log.Println("Успешное подключение к базе данных MySQL!")
}