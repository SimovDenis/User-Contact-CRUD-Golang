package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jaswdr/faker"

	_ "github.com/go-sql-driver/mysql"
)

const (
	username = "root"
	password = "password"
	hostname = "127.0.0.1:3307"
	dbname   = "gaussdb"

	userTable = `CREATE TABLE IF NOT EXISTS user(
		user_id int primary key auto_increment, 
		username varchar(50), 
        email varchar(50), 
		first_name varchar(50), 
		last_name varchar(50), 
		birth_year int(4), 
		password varchar(100),
		oib bigint(11)
		)`

	contactTable = `CREATE TABLE IF NOT EXISTS contact(
		user_id int primary key auto_increment, 
		first_name varchar(50),
	 	last_name varchar(50), 
		contact_number varchar(50), 
		email varchar(50)
		)`

	userContactTable = `CREATE TABLE IF NOT EXISTS user_contact(
		user_id int,
		contact_id int,
		PRIMARY KEY (user_id, contact_id),
		FOREIGN KEY (user_id) REFERENCES user(user_id),
		FOREIGN KEY (contact_id) REFERENCES contact(user_id)
	)`

	userCount        = 15
	contactCount     = 50
	userContactCount = 50
)

func dsn(dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, hostname, dbName)
}

func createDatabase() (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn(""))
	if err != nil {
		log.Printf("Error %s when opening DB\n", err)
		return nil, err
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	_, err = db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+dbname)
	if err != nil {
		log.Printf("Error %s when dropping DB\n", err)
		return nil, err
	}

	res, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname)
	if err != nil {
		log.Printf("Error %s when creating DB\n", err)
		return nil, err
	}
	no, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when fetching rows", err)
		return nil, err
	}
	log.Printf("rows affected: %d\n", no)

	db.Close()
	db, err = sql.Open("mysql", dsn(dbname))
	if err != nil {
		log.Printf("Error %s when opening DB", err)
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(time.Minute * 5)

	ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	err = db.PingContext(ctx)
	if err != nil {
		log.Printf("Errors %s pinging DB", err)
		return nil, err
	}
	log.Printf("Connected to DB %s successfully\n", dbname)
	return db, nil
}

func createTable(db *sql.DB, query string, tableName string) error {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when creating %s table", err, tableName)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when getting rows affected", err)
		return err
	}
	log.Printf("Rows affected when creating %s table: %d", tableName, rows)
	return nil
}

func createDatabaseSchema() {
	db, err := createDatabase()
	if err != nil {
		log.Printf("Error %s when getting db connection", err)
		return
	}
	defer db.Close()
	log.Printf("Successfully connected to database")

	err = createTable(db, userTable, "user")
	if err != nil {
		log.Printf("Create user table failed with error %s", err)
		return
	}

	err = createTable(db, contactTable, "contact")
	if err != nil {
		log.Printf("Create contact table failed with error %s", err)
		return
	}

	err = createTable(db, userContactTable, "user_contact")
	if err != nil {
		log.Printf("Create user_contact table failed with error %s", err)
		return
	}

	userIDs, err := insertUsers(db, userCount)
	if err != nil {
		log.Printf("Insert users failed with error %s", err)
		return
	}

	contactIDs, err := insertContacts(db, contactCount)
	if err != nil {
		log.Printf("Insert contacts failed with error %s", err)
		return
	}

	for i := 0; i < userContactCount; i++ {
		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)

		randomUserIDIndex := rng.Intn(len(userIDs))
		randomContactIDIndex := rng.Intn(len(contactIDs))

		row := db.QueryRow("SELECT COUNT(*) FROM user_contact WHERE user_id = ? AND contact_id = ?", userIDs[randomUserIDIndex], contactIDs[randomContactIDIndex])
		var count int
		err := row.Scan(&count)
		if err != nil {
			log.Printf("Failed to check if user_id and contact_id combination exists with error %s", err)
			return
		}

		if count == 0 {
			_, err := db.Exec("INSERT INTO user_contact(user_id, contact_id) VALUES (?, ?)", userIDs[randomUserIDIndex], contactIDs[randomContactIDIndex])
			if err != nil {
				log.Printf("Insert into user_contact failed with error %s", err)
				return
			}
		}
	}

}

func insertUsers(db *sql.DB, count int) ([]int64, error) {
	fake := faker.New()
	userIDs := []int64{}
	for i := 0; i < count; i++ {
		username := fake.Gamer().Tag()
		email := fake.Internet().Email()
		firstName := fake.Person().FirstName()
		lastName := fake.Person().LastName()
		birthYear := fake.Time().Year()
		password := fake.Internet().Password()
		oib := fake.RandomNumber(11)

		res, err := db.Exec("INSERT INTO user(username, email, first_name, last_name, birth_year, password, oib) VALUES (?, ?, ?, ?, ?, ?, ?)",
			username, email, firstName, lastName, birthYear, password, oib)
		if err != nil {
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, nil
}

func insertContacts(db *sql.DB, count int) ([]int64, error) {
	fake := faker.New()
	contactIDs := []int64{}
	for i := 0; i < count; i++ {
		email := fake.Internet().Email()
		firstName := fake.Person().FirstName()
		lastName := fake.Person().LastName()
		contact_number := fake.Person().Contact().Phone

		res, err := db.Exec("INSERT INTO contact(email, first_name, last_name, contact_number) VALUES (?, ?, ?, ?)",
			email, firstName, lastName, contact_number)
		if err != nil {
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		contactIDs = append(contactIDs, id)
	}
	return contactIDs, nil
}
