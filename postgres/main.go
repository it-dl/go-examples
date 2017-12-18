package main

import (
	"fmt"
	"github.com/go-ini/ini"
	"github.com/go-pg/pg"
	"os"
	"time"
)

type User struct {
	Id        int64
	Username  string    `sql:",notnull,unique"`
	Email     string    `sql:",notnull,unique"`
	CreatedAt time.Time `sql:",notnull"`
}

func (u User) String() string {
	return fmt.Sprintf("User<%d %s %s %s>", u.Id, u.Username, u.Email, u.CreatedAt)
}

func main() {
	db := pg.Connect(dbConnectOptions())
	defer db.Close()

	// Ignore errors from this call. (It's duplicate table error)
	_ = createSchema(db)

	switch os.Args[2] {
	case "add":
		if len(os.Args) != 5 {
			fatal("Usage: users add USERNAME EMAIL")
		}
		user, err := insert(db, os.Args[3], os.Args[4])
		checkErr(err)
		fmt.Printf("New user with id %d was created successfully\n", user.Id)
	case "del":
		if len(os.Args) < 4 {
			fatal("Usage: users del ID...")
		}
		deletedUsers, err := remove(db, os.Args[3:])
		checkErr(err)
		fmt.Printf("%d users were deleted successfully\n", deletedUsers)
	case "update":
		if len(os.Args) > 5 {
			fatal("Usage: users update ID EMAIL USERNAME")
		}
		user, err := update(db, os.Args[3], os.Args[4:])
		checkErr(err)
		fmt.Println(user)
	case "all":
		if len(os.Args) > 3 {
			fatal("Usage: users all")
		}
		users, err := all(db)
		checkErr(err)

		fmt.Println("id | username | email | created_at")

		for _, user := range users {
			fmt.Printf("%3v | %8v | %15v | %20v\n", user.Id, user.Username, user.Email, user.CreatedAt)
		}
	}
}

/**
 * Creates new user with passed username and email
 */
func insert(db *pg.DB, username, email string) (user User, err error) {
	user = User{
		Username:  username,
		Email:     email,
		CreatedAt: time.Now(),
	}
	err = db.Insert(&user)
	return
}

/**
 * Removes users with by ids
 */
func remove(db *pg.DB, ids []string) (int, error) {
	inIds := pg.In(ids)
	res, err := db.Model(&User{}).Where("id IN (?)", inIds).Delete()
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

/**
 * Updates user by id
 */
func update(db *pg.DB, id string, fields []string) (User, error) {
	var user User
	model := db.Model(&user)

	if len(fields) > 0 {
		model.Set("email = ?", fields[0])
	}

	if len(fields) > 1 {
		model.Set("username = ?", fields[1])
	}
	_, err := model.
		Where("id = ?", id).
		Returning("*").
		Update()

	if err != nil {
		return User{}, err
	}
	return user, nil
}

/**
 * Returns all users
 */
func all(db *pg.DB) ([]User, error) {
	var users []User
	err := db.Model(&users).Select()
	if err != nil {
		return []User{}, err
	}
	return users, nil
}

func createSchema(db *pg.DB) error {
	err := db.CreateTable(&User{}, nil)
	if err != nil {
		return err
	}
	return nil
}

/**
 * Parse database config .ini file and return connect options
 */
func dbConnectOptions() *pg.Options {
	cfg, err := ini.Load("db.ini")
	checkErr(err)

	section, err := cfg.GetSection("postgres")
	checkErr(err)

	dbname, err := section.GetKey("dbname")
	checkErr(err)

	user, err := section.GetKey("user")
	checkErr(err)

	password, err := section.GetKey("password")
	checkErr(err)

	addr := section.Key("addr").Validate(func(in string) string {
		if len(in) == 0 {
			return "localhost:5432"
		}
		return in
	})

	return &pg.Options{
		User:     user.String(),
		Password: password.String(),
		Database: dbname.String(),
		Addr:     addr,
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func fatal(v interface{}) {
	fmt.Println(v)
	os.Exit(1)
}
